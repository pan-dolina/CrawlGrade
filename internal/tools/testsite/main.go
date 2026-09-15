// Command testsite serves the CrawlGrade functional test site locally:
//
//	go run ./internal/tools/testsite -addr 127.0.0.1:8080
//	go run ./cmd/crawlgrade http://127.0.0.1:8080/ --allow-private
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pan-dolina/crawlgrade/internal/testsite"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address")
	flag.Parse()
	srv := &http.Server{
		Addr:              *addr,
		Handler:           testsite.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	fmt.Fprintf(os.Stderr, "test site on http://%s/\n", *addr)
	log.Fatal(srv.ListenAndServe())
}
