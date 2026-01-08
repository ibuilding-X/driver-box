package bacnet

import (
	"fmt"

	"github.com/ibuilding-x/driver-box/pkg/plugins/bacnet/internal/bacnet/btypes"
	"github.com/ibuilding-x/driver-box/pkg/plugins/bacnet/internal/bacnet/btypes/ndpu"
	"github.com/ibuilding-x/driver-box/pkg/plugins/bacnet/internal/bacnet/encoding"
)

/*
Is in beta, works but needs a decoder

in bacnet.Send() need to set the header.Function as btypes.BacFuncBroadcast

in bacnet.handleMsg() the npdu.IsNetworkLayerMessage is always rejected so this needs to be updated

*/

func (c *client) WhatIsNetworkNumber() (resp []*btypes.Address) {
	var err error
	dest := *c.dataLink.GetBroadcastAddress()
	enc := encoding.NewEncoder()
	npdu := &btypes.NPDU{
		Version:                 btypes.ProtocolVersion,
		Destination:             &dest,
		Source:                  c.dataLink.GetMyAddress(),
		IsNetworkLayerMessage:   true,
		NetworkLayerMessageType: ndpu.WhatIsNetworkNumber,
		// We are not expecting a direct reply from a single destination
		ExpectingReply: false,
		Priority:       btypes.Normal,
		HopCount:       btypes.DefaultHopCount,
	}
	enc.NPDU(npdu)
	// Run in parallel
	errChan := make(chan error)
	broadcast := &SetBroadcastType{Set: true, BacFunc: btypes.BacFuncBroadcast}
	go func() {
		_, err = c.Send(dest, npdu, enc.Bytes(), broadcast)
		errChan <- err
	}()
	values, err := c.utsm.Subscribe(1, 65534) //65534 is the max number a network can be
	if err != nil {
		fmt.Println(`err`, err)
	}
	err = <-errChan
	if err != nil {

	}

	for _, v := range values {
		r, ok := v.(btypes.NPDU)
		if r.Source != nil {
			resp = append(resp, r.Source)
		}
		if !ok {
			continue
		}
	}
	return resp

}
