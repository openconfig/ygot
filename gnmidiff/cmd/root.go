package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Execute() {
	rootCmd := &cobra.Command{
		Use:   "gnmidiff",
		Short: "gnmidiff is a utility for comparing between SetRequests and Notifications",
	}

	cfgFile := rootCmd.PersistentFlags().String("config_file", "", "Path to config file.")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if *cfgFile != "" {
			viper.SetConfigFile(*cfgFile)
			if err := viper.ReadInConfig(); err != nil {
				return fmt.Errorf("error reading config: %w", err)
			}
		}
		viper.BindPFlags(cmd.Flags())
		viper.AutomaticEnv()
		return nil
	}

	rootCmd.AddCommand(newSetRequestDiffCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
