package cmd

import (
	"fmt"

	"github.com/ibuilding-x/driver-box/v2/plugins/bacnet/internal/bacnet"
	"github.com/ibuilding-x/driver-box/v2/plugins/bacnet/internal/bacnet/btypes"
	"github.com/ibuilding-x/driver-box/v2/plugins/bacnet/internal/bacnet/helpers/data"
	ip2bytes "github.com/ibuilding-x/driver-box/v2/plugins/bacnet/internal/bacnet/helpers/ipbytes"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// readMultiCmd represents the readMultiCmd command
var readMultiCmd = &cobra.Command{
	Use:   "multi",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: readMulti,
}

func readMulti(cmd *cobra.Command, args []string) {
	if listProperties {
		btypes.PrintAllProperties()
		return
	}
	cb := &bacnet.ClientBuilder{
		Interface: Interface,
		Port:      Port,
	}
	c, _ := bacnet.NewClient(cb)
	defer c.Close()
	go c.ClientRun()

	ip, err := ip2bytes.New(deviceIP, uint16(devicePort))
	if err != nil {
		return
	}

	addr := btypes.Address{
		Net: uint16(networkNumber),
		Mac: ip,
		Adr: []uint8{uint8(deviceHardwareMac)},
	}

	object := btypes.ObjectID{
		Type:     btypes.DeviceType,
		Instance: btypes.ObjectInstance(deviceID),
	}

	//get max adpu len
	rp := btypes.PropertyData{
		Object: btypes.Object{
			ID: btypes.ObjectID{
				Type:     btypes.DeviceType,
				Instance: btypes.ObjectInstance(deviceID),
			},
			Properties: []btypes.Property{
				btypes.Property{
					Type:       btypes.PropMaxAPDU,
					ArrayIndex: bacnet.ArrayAll,
				},
			},
		},
	}

	dest := btypes.Device{
		ID:   object,
		Addr: addr,
	}
	// get the device MaxApdu
	out, err := c.ReadProperty(dest, rp)
	if err != nil {
		log.Fatal(err)
		return
	}

	_, dest.MaxApdu = data.ToUint32(out)

	fmt.Println("MaxApdu", dest.MaxApdu)

	//get object list
	rp = btypes.PropertyData{
		Object: btypes.Object{
			ID: btypes.ObjectID{
				Type:     8,
				Instance: btypes.ObjectInstance(deviceID),
			},
			Properties: []btypes.Property{
				btypes.Property{
					Type:       btypes.PropObjectList,
					ArrayIndex: bacnet.ArrayAll,
				},
			},
		},
	}

	// get the device object list
	out, err = c.ReadProperty(dest, rp)
	if err != nil {
		log.Fatal(err)
		return
	}

	rpm := btypes.MultiplePropertyData{}

	rpm.Objects = []btypes.Object{
		btypes.Object{
			ID: btypes.ObjectID{
				Type:     btypes.AnalogValue,
				Instance: 0,
			},
			Properties: []btypes.Property{
				{
					Type:       btypes.PropObjectName,
					ArrayIndex: bacnet.ArrayAll,
				},
			},
		},
		btypes.Object{
			ID: btypes.ObjectID{
				Type:     btypes.AnalogValue,
				Instance: 0,
			},
			Properties: []btypes.Property{
				{
					Type:       btypes.PropPresentValue,
					ArrayIndex: bacnet.ArrayAll,
				},
			},
		},
		btypes.Object{
			ID: btypes.ObjectID{
				Type:     btypes.AnalogValue,
				Instance: 1,
			},
			Properties: []btypes.Property{
				{
					Type:       btypes.PropObjectName,
					ArrayIndex: bacnet.ArrayAll,
				},
			},
		},
		btypes.Object{
			ID: btypes.ObjectID{
				Type:     btypes.AnalogValue,
				Instance: 1,
			},
			Properties: []btypes.Property{
				{
					Type:       btypes.PropPresentValue,
					ArrayIndex: bacnet.ArrayAll,
				},
			},
		},
	}

	//fmt.Println(rpm)
	rpmRes, err := c.ReadMultiProperty(dest, rpm)
	if err != nil {
		log.Println(err)
	}
	fmt.Println(rpmRes)

}

func init() {
	RootCmd.AddCommand(readMultiCmd)
	readMultiCmd.PersistentFlags().IntVarP(&deviceID, "device", "d", 1234, "device id")
	readMultiCmd.Flags().StringVarP(&deviceIP, "address", "", "192.168.15.202", "device ip")
	readMultiCmd.Flags().IntVarP(&devicePort, "dport", "", 47808, "device port")
	readMultiCmd.Flags().IntVarP(&networkNumber, "network", "", 0, "bacnet network number")
	readMultiCmd.Flags().IntVarP(&deviceHardwareMac, "mstp", "", 0, "device hardware mstp addr")

}
