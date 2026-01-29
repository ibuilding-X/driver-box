package bacnet

import (
	"fmt"
	"go/build"
	"os"
	"testing"

	pprint "github.com/ibuilding-x/driver-box/v2/plugins/bacnet/internal/bacnet/helpers/print"
)

var iface = "enp0s31f6"

func TestIam(t *testing.T) {

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	fmt.Println(gopath)

	cb := &ClientBuilder{
		Interface: iface,
	}
	c, _ := NewClient(cb)
	defer c.Close()
	go c.ClientRun()

	//resp := c.WhatIsNetworkNumber()

	resp := c.WhoIsRouterToNetwork()
	fmt.Println("WhoIsRouterToNetwork")
	pprint.Print(resp)

}
