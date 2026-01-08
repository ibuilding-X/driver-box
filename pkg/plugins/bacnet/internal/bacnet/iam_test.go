package bacnet

import (
	"fmt"
	pprint "github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet/helpers/print"
	"go/build"
	"os"
	"testing"
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
