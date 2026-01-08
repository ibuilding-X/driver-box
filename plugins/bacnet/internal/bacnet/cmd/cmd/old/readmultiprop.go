package cmd2

import (
	"fmt"
	"log"

	"github.com/ibuilding-x/driver-box/plugins/bacnet/internal/bacnet"
	"github.com/ibuilding-x/driver-box/plugins/bacnet/internal/bacnet/btypes"
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

var (
	deviceID       = 1
	listProperties = false
	startRange     = 1
	endRange       = 1
)

func readMulti(cmd *cobra.Command, args []string) {
	if listProperties {
		btypes.PrintAllProperties()
		return
	}
	cb := &bacnet.ClientBuilder{
		Interface: "",
		Port:      2,
	}
	c, _ := bacnet.NewClient(cb)
	defer c.Close()
	go c.ClientRun()
	wh := &bacnet.WhoIsOpts{}
	wh.Low = startRange
	wh.High = endRange
	// We need the actual address of the device first.
	resp, err := c.WhoIs(wh)
	if err != nil {
		log.Fatal(err)
	}

	if len(resp) == 0 {
		log.Fatal("Device id was not found on the network.")
	}

	for _, d := range resp {
		dest := d

		rp := btypes.PropertyData{
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

		out, err := c.ReadProperty(dest, rp)
		if err != nil {
			log.Fatal(err)
			return
		}
		ids, ok := out.Object.Properties[0].Data.([]interface{})
		if !ok {
			fmt.Println("unable to get object list")
			return
		}

		rpm := btypes.MultiplePropertyData{}
		rpm.Objects = make([]btypes.Object, len(ids))
		for i, raw_id := range ids {
			id, ok := raw_id.(btypes.ObjectID)
			if !ok {
				log.Printf("unable to read object id %v\n", raw_id)
				return
			}
			rpm.Objects[i].ID = id

			rpm.Objects[i].Properties = []btypes.Property{
				btypes.Property{
					Type:       btypes.PropObjectName,
					ArrayIndex: bacnet.ArrayAll,
				},
				btypes.Property{
					Type:       btypes.PropDescription,
					ArrayIndex: bacnet.ArrayAll,
				},
			}
		}

		x, err := c.ReadMultiProperty(dest, rpm)
		if err != nil {
			log.Println(err)
		}
		fmt.Println(x)
	}
}

func init() {
	//RootCmd.AddCommand(readMultiCmd)
	//readCmd.AddCommand(readMultiCmd)
	//readMultiCmd.Flags().IntVarP(&startRange, "start", "s", -1, "Start range of discovery")
	//readMultiCmd.Flags().IntVarP(&endRange, "end", "e", int(0xBAC0), "End range of discovery")

}
