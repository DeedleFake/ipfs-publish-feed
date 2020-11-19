package cli

import (
	"context"
	"os"
	"os/signal"
)

func SignalContext(ctx context.Context, signals ...os.Signal) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, signals...)
		defer signal.Stop(sig)

		<-sig
	}()

	return ctx
}
