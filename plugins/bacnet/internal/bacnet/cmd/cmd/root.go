package cmd

import (
	"fmt"
	"os"

	"github.com/ibuilding-x/driver-box/v2/plugins/bacnet/internal/bacnet/helpers/homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var Interface string
var Port int

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "baccli",
	Short: "description",
	Long:  `description`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.baccli.yaml)")
	RootCmd.PersistentFlags().StringVarP(&Interface, "interface", "i", "eth0", "Interface e.g. eth0")
	RootCmd.PersistentFlags().IntVarP(&Port, "port", "p", int(0xBAC0), "Port")

	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// We want to allow this to be accessed
	viper.BindPFlag("interface", RootCmd.PersistentFlags().Lookup("interface"))
	viper.BindPFlag("port", RootCmd.PersistentFlags().Lookup("port"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".baccli" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".baccli")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
