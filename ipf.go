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

// isContextError returns true if the given error is an error caused
// by a context.
func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// unsafeString performs an unsafe, but extremely efficient,
// conversion from a []byte to a string.
func unsafeString(data []byte) string {
	return *(*string)(unsafe.Pointer(&data))
}

func run(ctx context.Context) (err error) {
	addr := flag.String("addr", ":8080", "address to serve HTTP server on")
	api := flag.String("api", "http://localhost:5001", "base URL of HTTP API")
	topic := flag.String("topic", "publish", "pubsub topic to subscribe to")
	size := flag.Int("feedsize", 10, "maximum number of publishes to keep track of")
	flag.Parse()

	log.Println("Starting server...")
	defer func() {
		if err == nil {
			log.Println("Exiting...")
		}
	}()

	return Server{
		Address:  *addr,
		API:      *api,
		Topic:    *topic,
		FeedSize: *size,
	}.Serve(ctx)
}

func main() {
	ctx := cli.SignalContext(context.Background(), os.Interrupt)
	err := run(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
