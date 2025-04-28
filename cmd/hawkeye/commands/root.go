package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags
	cfgFile string

	// rootCmd represents the base command
	rootCmd = &cobra.Command{
		Use:   "hawkeye",
		Short: "A tool to monitor URLs for changes",
		Long: `Hawkeye is a powerful URL monitoring tool that helps you 
track changes in web content. Monitor multiple URLs
simultaneously and get notified when content changes.`,
		Run: func(cmd *cobra.Command, args []string) {
			// If no subcommand is provided, print help
			cmd.Help()
		},
	}
)

// Execute executes the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.hawkeye.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose output")

	// Add sub-commands
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(versionCmd)
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := getUserHomeDir()
		if err != nil {
			fmt.Println(err)
			return
		}

		// Search config in home directory with name ".hawkeye" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".hawkeye")
	}

	// Read in environment variables that match
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// getUserHomeDir returns the user's home directory
func getUserHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home, nil
}
