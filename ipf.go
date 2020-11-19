package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"unsafe"

	"github.com/DeedleFake/ipfs-publish-feed/internal/cli"
)

const (
	PublishTopic = "publish"
	WindowSize   = 10
)

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func unsafeString(data []byte) string {
	return *(*string)(unsafe.Pointer(&data))
}

func run(ctx context.Context) (err error) {
	addr := flag.String("addr", ":8080", "address to serve HTTP server on")
	api := flag.String("api", "http://localhost:5001", "base URL of HTTP API")
	flag.Parse()

	log.Println("Starting server...")
	defer func() {
		if err == nil {
			log.Println("Exiting...")
		}
	}()

	return Server{
		Address: *addr,
		API:     *api,
	}.Serve(ctx)
}

func main() {
	ctx := cli.SignalContext(context.Background(), os.Interrupt)
	err := run(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
