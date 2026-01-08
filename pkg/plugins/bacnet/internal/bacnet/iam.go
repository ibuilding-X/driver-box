package bacnet

import (
	"github.com/ibuilding-x/driver-box/pkg/plugins/bacnet/internal/bacnet/btypes"
	"github.com/ibuilding-x/driver-box/pkg/plugins/bacnet/internal/bacnet/encoding"
)

/*
not working
*/

func (c *client) IAm(dest btypes.Address, iam btypes.IAm) error {
	npdu := &btypes.NPDU{
		Version:     btypes.ProtocolVersion,
		Destination: &dest,
		//IsNetworkLayerMessage:   true,
		//NetworkLayerMessageType: 0x12,
		//Source:         c.dataLink.GetMyAddress(),
		ExpectingReply: false,
		Priority:       btypes.Normal,
		HopCount:       btypes.DefaultHopCount,
	}
	enc := encoding.NewEncoder()
	enc.NPDU(npdu)
	enc.IAm(iam)
	_, err := c.Send(dest, npdu, enc.Bytes(), nil)
	return err
}
