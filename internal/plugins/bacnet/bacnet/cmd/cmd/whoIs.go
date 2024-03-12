package cmd

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet/network"
	"time"

	pprint "github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet/helpers/print"

	"github.com/spf13/cobra"
)

// Flags
var startRange int
var endRange int

var outputFilename string

// whoIsCmd represents the whoIs command
var whoIsCmd = &cobra.Command{
	Use:   "whois",
	Short: "BACnet device discovery",
	Long: `whoIs does a bacnet network discovery to find devices in the network
 given the provided range.`,
	Run: main,
}

func main(cmd *cobra.Command, args []string) {

	client, err := network.New(&network.Network{Interface: Interface, Port: Port})
	if err != nil {
		fmt.Println("ERR-client", err)
		return
	}
	defer client.NetworkClose()
	go client.NetworkRun()

	if runDiscover {
		device, err := network.NewDevice(client, &network.Device{Ip: deviceIP, Port: Port})
		if err == nil {
			err = device.DeviceDiscover()
		}
		fmt.Println(err)
		return
	}

	wi := &bacnet.WhoIsOpts{
		High:            endRange,
		Low:             startRange,
		GlobalBroadcast: true,
		NetworkNumber:   uint16(networkNumber),
	}
	pprint.PrintJOSN(wi)

	fmt.Println("whois 1st")
	whoIs, err := client.Whois(wi)
	if err != nil {
		fmt.Println("ERR-whoIs", err)
		return
	}
	pprint.PrintJOSN(whoIs)

	time.Sleep(time.Second * 3)

	fmt.Println("whois 2nd")
	whoIs, err = client.Whois(wi)
	if err != nil {
		fmt.Println("ERR-whoIs", err)
		return
	}
	pprint.PrintJOSN(whoIs)

}

func init() {
	RootCmd.AddCommand(whoIsCmd)
	whoIsCmd.Flags().BoolVar(&runDiscover, "discover", false, "run network discover")
	whoIsCmd.Flags().IntVarP(&startRange, "start", "s", -1, "Start range of discovery")
	whoIsCmd.Flags().IntVarP(&endRange, "end", "e", -1, "End range of discovery")
	whoIsCmd.Flags().IntVarP(&networkNumber, "network", "", 0, "network number")
	whoIsCmd.Flags().StringVarP(&outputFilename, "out", "o", "", "Output results into the given filename in json structure.")
}
