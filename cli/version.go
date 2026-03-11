package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"

func newVersionCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput {
				payload := struct {
					Version string `json:"version"`
				}{Version: Version}

				encoded, err := json.Marshal(payload)
				if err != nil {
					return fmt.Errorf("marshal version output: %w", err)
				}

				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
				return err
			}

			_, err := fmt.Fprintf(cmd.OutOrStdout(), "sigil %s\n", Version)
			return err
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output version as JSON")

	return cmd
}
