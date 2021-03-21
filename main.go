package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/raspi/naapuri/pkg/neighbor"
	"github.com/raspi/naapuri/pkg/neighbor/parser"
	"os"
	"os/signal"
	"strings"
)

func OnEvent(evt parser.ChangeEvent) {
	fmt.Println(evt)
}

var (
	// These are set with Makefile -X=main.VERSION, etc
	VERSION   = `v0.0.0`
	BUILD     = `dev`
	BUILDDATE = `0000-00-00T00:00:00+00:00`
)

const (
	AUTHOR   = `Pekka Järvinen`
	YEAR     = 2021
	HOMEPAGE = `https://github.com/raspi/naapuri`
)

func main() {
	showVersionArg := flag.Bool(`V`, false, `Show version`)

	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stdout, `naapuri - show network mac address changes`+"\n")
		_, _ = fmt.Fprintf(os.Stdout, `Version %v (%v)`+"\n", VERSION, BUILDDATE)
		_, _ = fmt.Fprintf(os.Stdout, `(c) %v %v- [ %v ]`+"\n", AUTHOR, YEAR, HOMEPAGE)
		_, _ = fmt.Fprintf(os.Stdout, "\n")

		_, _ = fmt.Fprintf(os.Stdout, "Parameters:\n")

		paramMaxLen := 0

		flag.VisitAll(func(f *flag.Flag) {
			l := len(f.Name)
			if l > paramMaxLen {
				paramMaxLen = l
			}
		})

		flag.VisitAll(func(f *flag.Flag) {
			padding := strings.Repeat(` `, paramMaxLen-len(f.Name))
			_, _ = fmt.Fprintf(os.Stdout, "  -%s%s   %s   default: %q\n", f.Name, padding, f.Usage, f.DefValue)
		})

		_, _ = fmt.Fprintf(os.Stdout, "\n")
	}

	flag.Parse()

	if *showVersionArg {
		// Show version information
		_, _ = fmt.Fprintf(os.Stdout, `Version %s build %s built on %s`+"\n", VERSION, BUILD, BUILDDATE)
		os.Exit(0)
	}

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
