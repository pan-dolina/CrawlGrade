// Command crawlgrade audits the technical SEO and passive web hygiene of a
// website.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/pan-dolina/crawlgrade/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := cli.NewApp().Execute(ctx, os.Args[1:])
	stop()
	os.Exit(code)
}
