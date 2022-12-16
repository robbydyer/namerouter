package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	rootCmd := newRootCmd()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err.Error())

		os.Exit(1)
	}

	os.Exit(0)
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "namerouter",
		Short:        "Name based virtual host router",
		SilenceUsage: true,
	}
	f := rootCmd.PersistentFlags()

	viper.SetEnvPrefix("namerouter")
	if err := viper.BindPFlags(f); err != nil {
		fmt.Printf("Error binding pflags: %s\n", err.Error())
	}
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	rootCmd.AddCommand(
		newRunCmd(),
	)

	return rootCmd
}
