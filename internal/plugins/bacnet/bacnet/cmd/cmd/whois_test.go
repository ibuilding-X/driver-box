package cmd

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet"
	pprint "github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet/helpers/print"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet/network"
	"testing"
)

func TestByInterface(t *testing.T) {
	Interface = "VMware Network Adapter VMnet0"
	Port = 12345
	deviceIP = "192.168.9.19"
	client, err := network.New(&network.Network{Interface: Interface, Port: Port})
	if err != nil {
		fmt.Println("ERR-client", err)
		return
	}
	defer client.NetworkClose()
	go client.NetworkRun()

	_, err2 := network.New(&network.Network{Ip: "192.168.9.9", SubnetCIDR: 24, Port: Port})
	if err2 != nil {
		fmt.Println("ERR-client2", err2)
	}

	wi := &bacnet.WhoIsOpts{
		High:            -1,
		Low:             -1,
		GlobalBroadcast: true,
		NetworkNumber:   0,
	}
	pprint.PrintJOSN(wi)

	whoIs, err := client.Whois(wi)
	if err != nil {
		fmt.Println("ERR-whoIs", err)
		return
	}
	pprint.PrintJOSN(whoIs)
}

func TestByIp(t *testing.T) {
	// Interface = "VMware Network Adapter VMnet0"
	Port = 47808
	client, err := network.New(&network.Network{Ip: "192.168.9.9", SubnetCIDR: 24, Port: Port})
	if err != nil {
		fmt.Println("ERR-client", err)
		return
	}
	defer client.NetworkClose()
	go client.NetworkRun()

	wi := &bacnet.WhoIsOpts{
		High:            -1,
		Low:             -1,
		GlobalBroadcast: true,
		NetworkNumber:   0,
	}
	pprint.PrintJOSN(wi)

	whoIs, err := client.Whois(wi)
	if err != nil {
		fmt.Println("ERR-whoIs", err)
		return
	}
	pprint.PrintJOSN(whoIs)
}
