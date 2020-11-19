package cli

import (
	"context"
	"os"
	"os/signal"
)

// SignalContext returns a child context that is canceled when any one
// of the provided signals is received.
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
