// nolint:forbidigo,exhaustivestruct
package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/urfave/cli/v3"
)

var (
	version   = "unknown"
	buildDate = "unknown"
)

// parseCLI parses command lines arguments.
func parseCLI() error {
	appcmd := &cli.Command{
		Name:        "sheriff",
		Usage:       "Monitor multi services/processus",
		UsageText:   "sheriff [options]",
		Description: "Build: " + buildDate,
		Version:     version,
		CommandNotFound: func(c context.Context, cmd *cli.Command, name string) {
			fmt.Fprintf(os.Stderr, "Error. Unknown command: '%s'\n\n", name)
			cli.ShowAppHelpAndExit(cmd, 1)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Value:       "config.yaml",
				Usage:       "Configuration file",
				Aliases:     []string{"c"},
				Required:    false,
				DefaultText: "Default value is 'config.yaml'",
			},
		},
	}

	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Fprintln(os.Stdout, "Version:\t", cmd.Version)
	}

	appcmd.Action = action

	sort.Sort(cli.FlagsByName(appcmd.Flags))
	sort.Slice(appcmd.Commands, func(i, j int) bool {
		return appcmd.Commands[i].Name < appcmd.Commands[j].Name
	})

	if err := appcmd.Run(context.Background(), os.Args); err != nil {
		return fmt.Errorf("failed to parse command line arguments: %w", err)
	}

	return nil
}
