// Command loki-mock is a tiny Loki push API acceptor for architecture benches.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:3100", "listen address")
	flag.Parse()

	var posts atomic.Int64
	var bytes atomic.Int64
	var lines atomic.Int64

	mux := http.NewServeMux()
	mux.HandleFunc("/loki/api/v1/push", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		posts.Add(1)
		bytes.Add(int64(len(body)))
		var payload struct {
			Streams []struct {
				Values [][]string `json:"values"`
			} `json:"streams"`
		}
		_ = json.Unmarshal(body, &payload)
		for _, s := range payload.Streams {
			lines.Add(int64(len(s.Values)))
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"posts":%d,"bytes":%d,"lines":%d}`+"\n",
			posts.Load(), bytes.Load(), lines.Load())
	})

	srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	fmt.Fprintf(os.Stderr, "loki-mock listening on http://%s\n", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
