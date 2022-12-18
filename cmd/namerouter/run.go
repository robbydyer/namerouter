package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/robbydyer/namerouter/internal/namerouter"
)

type runCmd struct {
	configFile string
}

type cfg struct {
	Internal []string `yaml:"internal"`
	External []string `yaml:"external"`
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

	configData := map[string]*cfg{}

	data, err := os.ReadFile(r.configFile)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	nameHosts := []*namerouter.Namehost{}
	for ip, hostnames := range configData {
		nameHosts = append(nameHosts, &namerouter.Namehost{
			DestinationAddr: ip,
			ExternalHosts:   hostnames.External,
			InternalHosts:   hostnames.Internal,
		})
	}

	nr, err := namerouter.New(nameHosts...)
	if err != nil {
		return err
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		nr.Shutdown(stopCtx)
	}()

	return nr.Start()
}
