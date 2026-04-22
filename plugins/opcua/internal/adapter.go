package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
)

type opcuaClient struct {
	config *ConnectionConfig
	client *opcua.Client
	ctx    context.Context
	cancel context.CancelFunc
}

func newOpcuaClient(config *ConnectionConfig) (*opcuaClient, error) {
	ctx, cancel := context.WithCancel(context.Background())
	oc := &opcuaClient{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
	if err := oc.connect(); err != nil {
		cancel()
		return nil, err
	}
	return oc, nil
}

func (oc *opcuaClient) connect() error {
	endpoints, err := opcua.GetEndpoints(oc.ctx, oc.config.Endpoint)
	if err != nil {
		return fmt.Errorf("get endpoints error: %w", err)
	}
	var ep *ua.EndpointDescription
	for _, e := range endpoints {
		if oc.config.Policy != "" && e.SecurityPolicyURI != oc.config.Policy {
			continue
		}
		if oc.config.Mode != "" {
			modeMatch := false
			switch oc.config.Mode {
			case "None":
				modeMatch = e.SecurityMode == ua.MessageSecurityModeNone
			case "Sign":
				modeMatch = e.SecurityMode == ua.MessageSecurityModeSign
			case "SignAndEncrypt":
				modeMatch = e.SecurityMode == ua.MessageSecurityModeSignAndEncrypt
			}
			if !modeMatch {
				continue
			}
		}
		ep = e
		break
	}
	if ep == nil && len(endpoints) > 0 {
		ep = endpoints[0]
	}
	opts := []opcua.Option{
		opcua.SecurityPolicy(ep.SecurityPolicyURI),
		opcua.SecurityModeString(ep.SecurityMode.String()),
	}
	if oc.config.Username != "" {
		opts = append(opts, opcua.AuthUsername(oc.config.Username, oc.config.Password))
	}
	if oc.config.Timeout > 0 {
		opts = append(opts, opcua.DialTimeout(oc.config.Timeout))
	}
	opts = append(opts, opcua.ReconnectInterval(time.Second*5))
	opts = append(opts, opcua.AutoReconnect(true))
	client, err := opcua.NewClient(ep.EndpointURL, opts...)
	if err != nil {
		return fmt.Errorf("create opcua client error: %w", err)
	}
	if err := client.Connect(oc.ctx); err != nil {
		return fmt.Errorf("connect opcua server error: %w", err)
	}
	oc.client = client
	return nil
}

func (oc *opcuaClient) ReadNodes(nodeIds []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	if len(nodeIds) == 0 {
		return result, nil
	}
	nodesToRead := make([]*ua.ReadValueID, 0, len(nodeIds))
	for _, idStr := range nodeIds {
		id, err := ua.ParseNodeID(idStr)
		if err != nil {
			driverbox.Log().Warn("invalid nodeId", zap.String("nodeId", idStr), zap.Error(err))
			continue
		}
		nodesToRead = append(nodesToRead, &ua.ReadValueID{
			NodeID:      id,
			AttributeID: ua.AttributeIDValue,
		})
	}
	if len(nodesToRead) == 0 {
		return result, nil
	}
	req := &ua.ReadRequest{
		MaxAge:             2000,
		TimestampsToReturn: ua.TimestampsToReturnBoth,
		NodesToRead:        nodesToRead,
	}
	resp, err := oc.client.Read(oc.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	if resp.ResponseHeader.ServiceResult != ua.StatusOK {
		return nil, fmt.Errorf("read service error: %s", resp.ResponseHeader.ServiceResult)
	}
	for i, res := range resp.Results {
		if res.Status != ua.StatusOK {
			driverbox.Log().Warn("read node error", zap.String("nodeId", nodeIds[i]), zap.Any("status", res.Status))
			continue
		}
		result[nodeIds[i]] = res.Value.Value()
	}
	return result, nil
}

func (oc *opcuaClient) WriteNode(nodeId string, value interface{}) error {
	id, err := ua.ParseNodeID(nodeId)
	if err != nil {
		return fmt.Errorf("invalid nodeId: %w", err)
	}
	v, err := ua.NewVariant(value)
	if err != nil {
		return fmt.Errorf("create variant error: %w", err)
	}
	req := &ua.WriteRequest{
		NodesToWrite: []*ua.WriteValue{
			{
				NodeID:      id,
				AttributeID: ua.AttributeIDValue,
				Value: &ua.DataValue{
					Value: v,
				},
			},
		},
	}
	resp, err := oc.client.Write(oc.ctx, req)
	if err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	if resp.ResponseHeader.ServiceResult != ua.StatusOK {
		return fmt.Errorf("write service error: %s", resp.ResponseHeader.ServiceResult)
	}
	for _, status := range resp.Results {
		if status != ua.StatusOK {
			return fmt.Errorf("write node error: %s", status)
		}
	}
	return nil
}

func (oc *opcuaClient) Close() {
	if oc.client != nil {
		oc.client.Close(oc.ctx)
	}
	oc.cancel()
}
