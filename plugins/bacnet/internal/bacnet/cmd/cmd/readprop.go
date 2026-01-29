package cmd

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/ibuilding-x/driver-box/v2/plugins/bacnet/internal/bacnet"
	"github.com/ibuilding-x/driver-box/v2/plugins/bacnet/internal/bacnet/btypes"
	"github.com/ibuilding-x/driver-box/v2/plugins/bacnet/internal/bacnet/btypes/services"
	pprint "github.com/ibuilding-x/driver-box/v2/plugins/bacnet/internal/bacnet/helpers/print"
	"github.com/ibuilding-x/driver-box/v2/plugins/bacnet/internal/bacnet/network"
	"github.com/spf13/cobra"
)

// Flags
var (
	networkNumber     int
	deviceID          int
	deviceIP          string
	devicePort        int
	deviceHardwareMac int
	objectID          int
	objectType        int
	arrayIndex        uint32
	propertyType      string
	listProperties    bool
	segmentation      int
	maxADPU           int
	getDevicePoints   bool
)

// readCmd represents the read command
var readCmd = &cobra.Command{
	Use:   "read",
	Short: "Prints out a device's object's property",
	Long: `
 Given a device's object instance and selected property, we print the value
 stored there. There are some autocomplete features to try and minimize the
 amount of arguments that need to be passed, but do take into consideration
 this discovery process may cause longer reads.
	`,
	Run: readProp,
}

func readProp(cmd *cobra.Command, args []string) {

	localDevice, err := network.New(&network.Network{Interface: Interface, Port: Port})
	if err != nil {
		fmt.Println("ERR-client", err)
		return
	}
	defer localDevice.NetworkClose()
	go localDevice.NetworkRun()

	device, err := network.NewDevice(localDevice, &network.Device{Ip: deviceIP, DeviceID: deviceID, NetworkNumber: networkNumber, MacMSTP: deviceHardwareMac, MaxApdu: uint32(maxADPU), Segmentation: uint32(segmentation)})
	if err != nil {
		return
	}
	pprint.PrintJOSN(device)

	if getDevicePoints {
		points, err := device.GetDevicePoints(btypes.ObjectInstance(deviceID))
		if err != nil {
			return
		}
		for _, p := range points {
			fmt.Println("pnt----------pnt----------", p.Name)
			pprint.PrintJOSN(p)
			//fmt.Println(p.ObjectType)
		}
		return
	}

	var propInt btypes.PropertyType
	// Check to see if an int was passed
	if i, err := strconv.Atoi(propertyType); err == nil {
		propInt = btypes.PropertyType(uint32(i))
	} else {
		propInt, err = btypes.Get(propertyType)
	}

	obj := &network.Object{
		ObjectID:   btypes.ObjectInstance(objectID),
		ObjectType: btypes.ObjectType(objectType),
		Prop:       propInt,
		ArrayIndex: arrayIndex, //btypes.ArrayAll
	}
	read, err := device.Read(obj)
	pprint.PrintJOSN(read)
	fmt.Println(read.Object.Properties[0].Data)

	arr := read.Object.Properties[0].Data.(*btypes.BitString)
	for i, aa := range arr.GetValue() {
		fmt.Println(i, aa)
		fmt.Println("TYPE", reflect.TypeOf(aa))
	}
	fmt.Println("TYPE", reflect.TypeOf(read.Object.Properties[0].Data))
	//fmt.Println(1111, arr)
	//for i, a := range arr {
	//	fmt.Println(i, a)
	//}
	ss := services.Supported{}
	ss.ListAll()
}
func init() {
	// Descriptions are kept separate for legibility purposes.
	propertyTypeDescr := `type of read that will be done. Support both the
	property type as an integer or as a string. e.g. PropObjectName or 77 are both
	support. Run --list to see available properties.`
	listPropertiesDescr := `list all string versions of properties that are
	support by property flag`

	RootCmd.AddCommand(readCmd)

	// Pass flags to children
	readCmd.PersistentFlags().IntVarP(&deviceID, "device", "", 0, "device id")
	readCmd.Flags().StringVarP(&deviceIP, "address", "", "192.168.15.202", "device ip")
	readCmd.Flags().IntVarP(&devicePort, "dport", "", 47808, "device port")
	readCmd.Flags().IntVarP(&networkNumber, "network", "", 0, "bacnet network number")
	readCmd.Flags().IntVarP(&deviceHardwareMac, "mstp", "", 0, "device hardware mstp addr")
	readCmd.Flags().IntVarP(&maxADPU, "adpu", "", 0, "device max adpu")
	readCmd.Flags().IntVarP(&segmentation, "seg", "", 0, "device segmentation")
	readCmd.Flags().IntVarP(&objectID, "objectID", "", -1, "object ID")
	readCmd.Flags().IntVarP(&objectType, "objectType", "", 8, "object type")
	readCmd.Flags().StringVarP(&propertyType, "property", "", btypes.ObjectNameStr, propertyTypeDescr)
	readCmd.Flags().Uint32Var(&arrayIndex, "index", bacnet.ArrayAll, "Which position to return.")

	readCmd.PersistentFlags().BoolVarP(&listProperties, "list", "l", false, listPropertiesDescr)

	readCmd.PersistentFlags().BoolVarP(&getDevicePoints, "device-points", "", false, "get device points list")
}
