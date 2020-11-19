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
)

type Server struct {
	Address string
	API     string
}

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
		ctx, _ = context.WithTimeout(context.Background(), time.Minute)
		err := server.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		return nil

	case err := <-errs:
		return err
	}
}

func (s Server) handler(ctx context.Context) http.Handler {
	data := make(chan []string)
	go func() {
		window := make([]string, 0, 10)

		incoming := make(chan PubSubData)
		go Subscribe(ctx, incoming, s.API, PublishTopic)

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
				log.Printf("Publish: %q", data)

				window = append(window, unsafeString(data))
				if len(window) > WindowSize {
					copy(window[:WindowSize], window[len(window)-WindowSize:])
					window = window[:WindowSize]
				}

			case data <- append([]string(nil), window...):
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

var feedTmpl = template.Must(template.New("atom").Parse(`
<?xml version="1.0" encoding="utf-8"?>

<feed xmlns="http://www.w3.org/2005/Atom">
	<title>IPFS Publish Feed</title>

	{{range .Data}}
		<entry>
			<title>{{.}}</title>
			<link href="{{printf "https://ipfs.io/ipfs/%v" .}}" />
		</entry>
	{{end}}
</feed>
`))
