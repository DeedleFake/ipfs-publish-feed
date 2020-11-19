package main

import (
	"context"
	"log"
	"os"

	"github.com/DeedleFake/ipfs-publish-feed/internal/cli"
)

func run(ctx context.Context) error {
	return nil
}

func main() {
	ctx := cli.SignalContext(context.Background(), os.Interrupt)
	err := run(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
