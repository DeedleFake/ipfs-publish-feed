package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"text/template"
	"time"

	"github.com/ipfs/go-cid"
)

// Server is listens to the stream of incoming data and serves the
// feed.
type Server struct {
	// Address is the address for the HTTP server to listen on.
	Address string

	// API is the base URL of the IPFS HTTP API.
	API string

	// Topic is the pubsub topic to listen for new publishes on.
	Topic string

	// FeedSize is the maximum number of publishes to keep track of at a
	// time.
	FeedSize int
}

// Serve runs the server, listening to the stream of incoming data
// from IPFS and serving a feed of it over HTTP.
func (s Server) Serve(ctx context.Context) error {
	server := http.Server{
		Addr:    s.Address,
		Handler: s.handler(ctx),
		BaseContext: func(lis net.Listener) context.Context {
			return ctx
		},
	}

	errs := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			errs <- fmt.Errorf("start HTTP server: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		ctx, _ := context.WithTimeout(context.Background(), time.Minute)
		err := server.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		return nil

	case err := <-errs:
		return err
	}
}

// handler returns the handler for the HTTP server.
func (s Server) handler(ctx context.Context) http.Handler {
	data := make(chan []FileStat)
	go func() {
		window := make([]FileStat, 0, s.FeedSize+1)

		incoming := make(chan PubSubData)
		go Subscribe(ctx, incoming, s.API, s.Topic)

		stat := make(chan FileStat)

		for {
			select {
			case <-ctx.Done():
				return

			case next := <-incoming:
				data, err := base64.StdEncoding.DecodeString(next.Data)
				if err != nil {
					log.Printf("Error: decode data: %v", err)
					continue
				}
				cid, err := cid.Decode(unsafeString(data))
				if err != nil {
					log.Printf("Error: decode CID: %v", err)
					continue
				}
				log.Printf("Publish: %q", cid)

				go func() {
					info, err := Stat(ctx, s.API, cid)
					if err != nil {
						log.Printf("Error: stat %q: %v", cid, err)
						return
					}
					stat <- info
				}()

			case info := <-stat:
				window = append(window, info)
				if len(window) > s.FeedSize {
					copy(window[:s.FeedSize], window[len(window)-s.FeedSize:])
					window = window[:s.FeedSize]
				}

			case data <- append([]FileStat(nil), window...):
			}
		}
	}()

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		log.Printf("%v %q", req.Method, req.URL)

		rw.Header().Set("Content-Type", "application/atom+xml")
		err := feedTmpl.Execute(rw, map[string]interface{}{
			"Data": <-data,
		})
		if err != nil {
			log.Printf("Error: execute template: %v", err)
			return
		}
	})
}

// feedTmpl is the template for the feed.
var feedTmpl = template.Must(template.New("atom").Parse(`
<?xml version="1.0" encoding="utf-8"?>

<feed xmlns="http://www.w3.org/2005/Atom">
	<title>IPFS Publish Feed</title>

	{{range .Data}}
		<entry>
			<title>{{.}}</title>
			<summary>Type: {{.Type}} Size: {{.CumulativeSize}}</summary>
			<link href="{{printf "https://ipfs.io/ipfs/%v" .}}" />
		</entry>
	{{end}}
</feed>
`))
