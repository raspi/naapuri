package main

import (
	"context"
	"fmt"
	"github.com/raspi/naapuri/pkg/neighbor"
	"github.com/raspi/naapuri/pkg/neighbor/parser"
	"os"
	"os/signal"
)

func OnEvent(evt parser.ChangeEvent) {
	fmt.Println(evt)
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill)

	defer func(sig chan os.Signal, c context.CancelFunc) {
		signal.Stop(sig)
		c()
	}(signalChan, cancel)

	go func(sig chan os.Signal, c context.CancelFunc) {
		select {
		case <-sig: // first signal, cancel context
			c()
		case <-ctx.Done():
		}

		<-sig // second signal, hard exit
		os.Exit(0)
	}(signalChan, cancel)

	l, err := neighbor.New(OnEvent, ctx)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, `error: %v`, err)
		os.Exit(1)
	}
	defer l.Close()

	// Errors while listening on changes
	errs := make(chan error)
	defer close(errs)

	go func(e chan error) {
		// Start listening
		l.Listen(e)
	}(errs)

	for e := range errs {
		// Print errors during parsing

		if e == context.Canceled {
			break
		}

		_, _ = fmt.Fprintf(os.Stderr, `error: %v`+"\n", e)
	}

}
