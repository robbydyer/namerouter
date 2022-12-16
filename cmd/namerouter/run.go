package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/robbydyer/namerouter/internal/namerouter"
)

type runCmd struct {
	configFile string
}

func newRunCmd() *cobra.Command {
	r := &runCmd{}

	cmd := &cobra.Command{
		Use:  "run",
		RunE: r.run,
	}

	f := cmd.Flags()

	f.StringVar(&r.configFile, "config-file", "", "config file name")

	return cmd
}

func (r *runCmd) run(cmd *cobra.Command, args []string) error {
	if r.configFile == "" {
		return fmt.Errorf("missing --config-file")
	}

	nr, err := namerouter.New()
	if err != nil {
		return err
	}

	configData := map[string][]string{}

	data, err := os.ReadFile(r.configFile)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	for ip, hostnames := range configData {
		nr.AddNamehost(&namerouter.Namehost{
			DestinationAddr: ip,
			Hosts:           hostnames,
		})
	}

	return nr.Start()
}
