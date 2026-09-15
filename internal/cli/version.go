package cli

import (
	"encoding/json"
	"fmt"

	"github.com/pan-dolina/crawlgrade/internal/version"
	"github.com/spf13/cobra"
)

func (a *App) newVersionCommand() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := version.Get()
			if asJSON {
				enc := json.NewEncoder(a.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}
			commit := info.Commit
			if len(commit) > 12 {
				commit = commit[:12]
			}
			if commit == "" {
				commit = "unknown"
			}
			_, err := fmt.Fprintf(a.Stdout, "crawlgrade %s (commit %s, %s, %s)\n", info.Version, commit, info.GoVersion, info.Platform)
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print version information as JSON")
	return cmd
}
