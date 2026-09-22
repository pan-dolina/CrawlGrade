package cli

import (
	"encoding/json"
	"github.com/pan-dolina/crawlgrade/internal/report"
	"github.com/spf13/cobra"
	"strings"
)

func (a *App) newDiffCommand() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{Use: "diff BASELINE CURRENT", Short: "Compare two saved JSON reports", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		load := func(path string) (*report.Report, error) {
			b, err := readBaseline(path)
			if err != nil {
				return nil, err
			}
			return report.Load(b)
		}
		base, err := load(args[0])
		if err != nil {
			return err
		}
		current, err := load(args[1])
		if err != nil {
			return err
		}
		d, err := report.Compare(current, base)
		if err != nil {
			return err
		}
		if asJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(d)
		}
		var b strings.Builder
		d.RenderTerminal(&b)
		_, err = cmd.OutOrStdout().Write([]byte(report.TerminalText(b.String())))
		return err
	}}
	c.Flags().BoolVar(&asJSON, "json", false, "write the diff as JSON")
	return c
}
