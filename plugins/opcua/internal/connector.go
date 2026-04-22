package internal

import (
	"errors"
	"fmt"
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
	"github.com/ibuilding-x/driver-box/v2/pkg/convutil"
	"go.uber.org/zap"
)

func newConnector(p *Plugin, config *ConnectionConfig) (*connector, error) {
	conn := &connector{
		config:  config,
		plugin:  p,
		nodes:   make(map[string]*NodeConfig),
		virtual: config.Virtual,
	}
	if !config.Virtual {
		client, err := newOpcuaClient(config)
		if err != nil {
			return nil, fmt.Errorf("create opcua client error: %w", err)
		}
		conn.client = client
	}
	return conn, nil
}

func (c *connector) createNodeGroups(model config.DeviceModel, device config.Device) {
	for _, point := range model.DevicePoints {
		nodeCfg := &NodeConfig{}
		ext, _ := point.FieldValue("ext")
		if ext != nil {
			if err := convutil.Struct(ext, nodeCfg); err != nil {
				driverbox.Log().Error("parse node config error", zap.String("point", point.Name()), zap.Error(err))
				continue
			}
		}
		if nodeCfg.NodeId == "" {
			driverbox.Log().Warn("point has no nodeId", zap.String("point", point.Name()))
			continue
		}
		nodeCfg.PointName = point.Name()
		c.nodes[point.Name()] = nodeCfg
	}
}

func (c *connector) initCollectTask(config *ConnectionConfig) (interface{}, error) {
	interval := config.Interval
	if interval <= 0 {
		interval = time.Second * 10
	}
	return driverbox.AddFunc(fmt.Sprintf("%ds", int(interval.Seconds())), func() {
		c.collectData()
	})
}

func (c *connector) collectData() {
	if c.close {
		return
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.virtual {
		c.collectMockData()
		return
	}
	if c.client == nil {
		driverbox.Log().Error("opcua client is nil")
		return
	}
	values, err := c.client.ReadNodes(c.getNodeIds())
	if err != nil {
		driverbox.Log().Error("read opcua nodes error", zap.Error(err))
		return
	}
	deviceData := c.processReadValues(values)
	if len(deviceData.Values) > 0 {
		driverbox.Export([]plugin.DeviceData{deviceData})
	}
}

func (c *connector) getNodeIds() []string {
	nodeIds := make([]string, 0, len(c.nodes))
	for _, node := range c.nodes {
		nodeIds = append(nodeIds, node.NodeId)
	}
	return nodeIds
}

func (c *connector) processReadValues(values map[string]interface{}) plugin.DeviceData {
	pointData := make([]plugin.PointData, 0, len(values))
	for pointName, node := range c.nodes {
		if value, ok := values[node.NodeId]; ok {
			if node.Scale != 0 && node.Scale != 1 {
				switch v := value.(type) {
				case float64:
					value = v * node.Scale
				case int:
					value = float64(v) * node.Scale
				case int64:
					value = float64(v) * node.Scale
				}
			}
			pointData = append(pointData, plugin.PointData{
				PointName: pointName,
				Value:     value,
			})
		}
	}
	return plugin.DeviceData{
		ID:         c.config.ConnectionKey,
		Values:     pointData,
		ExportType: plugin.RealTimeExport,
	}
}

func (c *connector) collectMockData() {
	pointData := make([]plugin.PointData, 0, len(c.nodes))
	for pointName := range c.nodes {
		pointData = append(pointData, plugin.PointData{
			PointName: pointName,
			Value:     100,
		})
	}
	deviceData := plugin.DeviceData{
		ID:         c.config.ConnectionKey,
		Values:     pointData,
		ExportType: plugin.RealTimeExport,
	}
	driverbox.Export([]plugin.DeviceData{deviceData})
}

func (c *connector) Encode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	if mode == plugin.ReadMode {
		nodeIds := make([]string, 0, len(values))
		for _, v := range values {
			if node, ok := c.nodes[v.PointName]; ok {
				nodeIds = append(nodeIds, node.NodeId)
			}
		}
		return &ReadRequest{Nodes: nodeIds}, nil
	}
	if mode == plugin.WriteMode {
		if len(values) == 0 {
			return nil, errors.New("no write values")
		}
		writeReqs := make([]*WriteRequest, 0, len(values))
		for _, v := range values {
			if node, ok := c.nodes[v.PointName]; ok {
				if !node.Writeable {
					driverbox.Log().Warn("point is not writeable", zap.String("point", v.PointName))
					continue
				}
				writeReqs = append(writeReqs, &WriteRequest{
					NodeId: node.NodeId,
					Value:  v.Value,
				})
			}
		}
		return writeReqs, nil
	}
	return nil, plugin.NotSupportEncode
}

func (c *connector) Send(data interface{}) (err error) {
	if c.virtual {
		return nil
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.client == nil {
		return errors.New("opcua client is nil")
	}
	switch req := data.(type) {
	case *ReadRequest:
		_, err := c.client.ReadNodes(req.Nodes)
		return err
	case []*WriteRequest:
		for _, wr := range req {
			if err := c.client.WriteNode(wr.NodeId, wr.Value); err != nil {
				driverbox.Log().Error("write node error", zap.String("nodeId", wr.NodeId), zap.Error(err))
			}
		}
		return nil
	default:
		return errors.New("unsupported data type")
	}
}

func (c *connector) Release() (err error) {
	return c.Close()
}

func (c *connector) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.close = true
	if c.client != nil {
		c.client.Close()
	}
	return nil
}
