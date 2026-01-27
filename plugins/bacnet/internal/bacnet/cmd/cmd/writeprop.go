package cmd

import (
	"fmt"
	"strconv"

	"github.com/ibuilding-x/driver-box/plugins/bacnet/internal/bacnet"
	"github.com/ibuilding-x/driver-box/plugins/bacnet/internal/bacnet/btypes"
	"github.com/ibuilding-x/driver-box/plugins/bacnet/internal/bacnet/btypes/null"
	ip2bytes "github.com/ibuilding-x/driver-box/plugins/bacnet/internal/bacnet/helpers/ipbytes"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// write represents the write command
var writeCmd = &cobra.Command{
	Use:   "write",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: writeProp,
}

// Flags
var (
	targetValue string
	priority    uint
	isNull      bool
)

func init() {
	// Descriptions are kept separate for legibility purposes.
	propertyTypeDescr := `type of read that will be done. Support both the
	property type as an integer or as a string. e.g. PropObjectName or 77 are both
	support. Run --list to see available properties.`
	listPropertiesDescr := `list all string versions of properties that are
	support by property flag`

	RootCmd.AddCommand(writeCmd)
	// Pass flags to children
	writeCmd.PersistentFlags().IntVarP(&deviceID, "device", "d", 1234, "device id")
	writeCmd.Flags().StringVarP(&deviceIP, "address", "", "192.168.15.202", "device ip")
	writeCmd.Flags().IntVarP(&devicePort, "dport", "", 47808, "device port")
	writeCmd.Flags().IntVarP(&networkNumber, "network", "", 0, "bacnet network number")
	writeCmd.Flags().IntVarP(&deviceHardwareMac, "mstp", "", 0, "device hardware mstp addr")
	writeCmd.Flags().IntVarP(&objectID, "objectID", "o", 1234, "object ID")
	writeCmd.Flags().IntVarP(&objectType, "objectType", "j", 8, "object type")
	writeCmd.Flags().StringVarP(&propertyType, "property", "t",
		btypes.ObjectNameStr, propertyTypeDescr)
	writeCmd.Flags().StringVarP(&targetValue, "value", "v",
		"", "value that will be set")

	writeCmd.Flags().UintVar(&priority, "priority", 0, "default is the lowest priority")
	writeCmd.Flags().Uint32Var(&arrayIndex, "index", bacnet.ArrayAll, "Which position to return.")
	writeCmd.PersistentFlags().BoolVarP(&listProperties, "list", "l", false,
		listPropertiesDescr)

	writeCmd.PersistentFlags().BoolVar(&isNull, "null", false,
		"clear value by writting null to it.")
}

func writeProp(cmd *cobra.Command, args []string) {
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
		Type:     8,
		Instance: 88, //1103,
	}

	dest := btypes.Device{
		ID:   object,
		Addr: addr,
	}

	var propInt btypes.PropertyType
	// Check to see if an int was passed
	if i, err := strconv.Atoi(propertyType); err == nil {
		propInt = btypes.PropertyType(uint32(i))
	} else {
		propInt, err = btypes.Get(propertyType)
	}

	if btypes.IsDeviceProperty(propInt) {
		objectType = 8
	}

	if err != nil {
		log.Fatal(err)
	}

	rp := btypes.PropertyData{
		Object: btypes.Object{
			ID: btypes.ObjectID{
				Type:     btypes.ObjectType(objectType),
				Instance: btypes.ObjectInstance(objectID),
			},
			Properties: []btypes.Property{
				{
					Type:       propInt,
					ArrayIndex: arrayIndex,
					Priority:   btypes.NPDUPriority(priority),
				},
			},
		},
	}

	var wp interface{}
	if isNull {
		wp = null.Null{}
	} else {
		out, err := c.ReadProperty(dest, rp)
		if err != nil {
			if rp.Object.Properties[0].Type == btypes.PropObjectList {
				log.Error("Note: PropObjectList reads may need to be broken up into multiple reads due to length. Read index 0 for array length")
			}
			log.Fatal(err)
		}
		if len(out.Object.Properties) == 0 {
			fmt.Println("No value returned")
			return
		}

		rd := out.Object.Properties[0].Data
		log.Infof("Current value %v, type %T", rd, rd)

		if targetValue == "" {
			log.Fatal("nothing was written")
			return
		}

		switch rd.(type) {
		case uint32:
			var f float64
			f, err = strconv.ParseFloat(targetValue, 32)
			wp = uint32(f)
		case float32:
			var f float64
			f, err = strconv.ParseFloat(targetValue, 32)
			wp = float32(f)
		case float64:
			wp, err = strconv.ParseFloat(targetValue, 64)
		case string:
			wp = targetValue
		default:
			err = fmt.Errorf("unable to handle a type %T", rd)
		}
		if err != nil {
			log.Printf("Expects a %T", rp.Object.Properties[0].Data)
		}
	}

	rp.Object.Properties[0].Data = wp
	// log.Printf("Writting: %v", wp)
	log.Printf("writting to: %v, value: \n", dest, rp)
	err = c.WriteProperty(dest, rp)
	if err != nil {
		log.Println(err)
	}
}
