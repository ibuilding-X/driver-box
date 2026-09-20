package internal

import (
	"errors"
	"fmt"
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
	protocol "github.com/ibuilding-x/driver-box/v2/pkg/iec104"
	"github.com/orglibs/go-iecp5/asdu"
)

// operation 是预编码的单个阶段；选择/执行分别作为一个阶段等待确认。
type operation struct {
	// frame 为完整 ASDU，Encode 只构建报文，不执行网络写入。
	frame *asdu.ASDU
	// confirm 为 true 时等待匹配的 ACT_CON；读命令响应通过上报通道异步处理。
	confirm bool
}

// request 绑定创建它的连接，避免把一个站的地址报文发送到另一个站。
type request struct {
	owner      *connector  // 编码所属连接，Send 校验身份。
	operations []operation // 保留调用方点位顺序，以及每个点的选择/执行顺序。
}

// Encode 检查设备、点名、权限和值域，并在发送任何报文前完成整批编码。
func (c *connector) Encode(deviceID string, mode plugin.EncodeMode, values ...plugin.PointData) (any, error) {
	if c.ctx.Err() != nil {
		return nil, errors.New("IEC104 connection closed")
	}
	if mode != plugin.ReadMode && mode != plugin.WriteMode {
		return nil, plugin.NotSupportEncode
	}
	points, ok := c.nodes[deviceID]
	if !ok {
		return nil, fmt.Errorf("unknown IEC104 device %s", deviceID)
	}
	if len(values) == 0 {
		return nil, errors.New("no IEC104 points requested")
	}
	req := &request{owner: c}
	for _, value := range values {
		n, ok := points[value.PointName]
		if !ok {
			return nil, fmt.Errorf("unknown point %s/%s", deviceID, value.PointName)
		}
		if mode == plugin.ReadMode {
			if n.access == config.ReadWrite_W {
				return nil, fmt.Errorf("point %s is write-only", n.name)
			}
			a := asdu.NewASDU(&c.settings.params, asdu.Identifier{Type: asdu.C_RD_NA_1, Variable: asdu.VariableStruct{Number: 1}, Coa: asdu.CauseOfTransmission{Cause: asdu.Request}, CommonAddr: asdu.CommonAddr(n.CommonAddress), OrigAddr: c.settings.params.OrigAddress})
			if err := a.AppendInfoObjAddr(asdu.InfoObjAddr(n.IOA)); err != nil {
				return nil, err
			}
			req.operations = append(req.operations, operation{frame: a})
		} else {
			if n.access == config.ReadWrite_R {
				return nil, fmt.Errorf("point %s is read-only", n.name)
			}
			if n.SelectBeforeExecute {
				a, err := protocol.Command(&c.settings.params, n.Point, value.Value, true)
				if err != nil {
					return nil, fmt.Errorf("point %s: %w", n.name, err)
				}
				req.operations = append(req.operations, operation{a, true})
			}
			a, err := protocol.Command(&c.settings.params, n.Point, value.Value, false)
			if err != nil {
				return nil, fmt.Errorf("point %s: %w", n.name, err)
			}
			req.operations = append(req.operations, operation{a, true})
		}
	}
	return req, nil
}

// Send 对控制命令等待激活确认，对读取命令只确认发送入队。
// 多点操作不是事务：若后续点失败，前面已确认的点不会回滚。
func (c *connector) Send(data any) error {
	req, ok := data.(*request)
	if !ok || req == nil || req.owner != c || len(req.operations) == 0 {
		return errors.New("invalid IEC104 request")
	}
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	c.mu.Lock()
	generation := c.generation
	blocked := c.blocked && c.blockedGeneration == generation
	c.mu.Unlock()
	if blocked {
		return errors.New("IEC104 session has an unconfirmed command; waiting for reconnect")
	}
	for _, op := range req.operations {
		if c.ctx.Err() != nil {
			return errors.New("IEC104 connection closed")
		}
		c.mu.Lock()
		changed := generation != c.generation
		c.mu.Unlock()
		if changed || !c.client.IsActive() {
			return errors.New("IEC104 data transfer is not active")
		}
		if !op.confirm {
			if err := c.client.Send(op.frame); err != nil {
				return err
			}
			continue
		}
		if err := c.command(op.frame); err != nil {
			return err
		}
	}
	return nil
}

func (c *connector) command(frame *asdu.ASDU) error {
	pending := &pendingCommand{frame: frame, result: make(chan error, 1)}
	c.mu.Lock()
	c.pending = pending
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		if c.pending == pending {
			c.pending = nil
		}
		c.mu.Unlock()
	}()
	if err := c.client.Send(frame); err != nil {
		return err
	}
	timer := time.NewTimer(c.settings.commandTimeout)
	defer timer.Stop()
	select {
	case err := <-pending.result:
		return err
	case <-c.ctx.Done():
		return errors.New("IEC104 connection closed")
	case <-timer.C:
		// IEC104 没有业务事务号；超时后关闭当前会话，避免迟到确认误配
		// 后续同地址同值命令。执行结果未知，不做自动重发。
		c.mu.Lock()
		c.blocked = true
		c.blockedGeneration = c.generation
		c.mu.Unlock()
		c.client.Disconnect()
		return errors.New("IEC104 activation confirmation timeout; execution outcome unknown")
	}
}

func (c *connector) failPending(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pending != nil {
		select {
		case c.pending.result <- err:
		default:
		}
	}
}

// Release 只结束框架的一次调用，不关闭共享长连接。核心读写路径每次都会调用它；
// 会话由插件持有，必须通过 Destroy/Close 统一释放。
func (c *connector) Release() error { return nil }

// Close 取消等待并等待传输、维护和上报任务退出，保证热重载后旧连接不再上报。
func (c *connector) Close() error {
	c.closeOnce.Do(func() {
		c.cancel()
		c.failPending(errors.New("IEC104 connection closed"))
		_ = c.client.Close()
		c.client.Wait()
		<-c.done
		<-c.exportDone
	})
	return nil
}
