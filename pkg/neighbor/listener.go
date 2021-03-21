package neighbor

import (
	"bytes"
	"context"
	"fmt"
	"github.com/raspi/naapuri/pkg/neighbor/parser"
	"golang.org/x/sys/unix"
	"os"
	"syscall"
)

type OnEventFunc func(evt parser.ChangeEvent)

type Listener struct {
	onEventFunc OnEventFunc // This function gets called when change events happen
	fd          int         // file handle for reading netlink messages
	ctx         context.Context
}

func New(evt OnEventFunc, ctx context.Context) (l Listener, err error) {
	// Open
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_DGRAM, unix.NETLINK_ROUTE)
	if err != nil {
		return l, err
	}

	err = unix.Bind(fd, &unix.SockaddrNetlink{
		//Family: unix.AF_NETLINK,
		Family: unix.AF_ROUTE,
		Groups: unix.RTMGRP_NEIGH,
		Pid:    0,
	})

	if err != nil {
		return l, err
	}

	return Listener{
		fd:          fd,
		onEventFunc: evt,
		ctx:         ctx,
	}, nil
}

func (listener *Listener) Listen(errch chan error) {
	buffer := make([]byte, os.Getpagesize())

	for {
		for {
			select {
			case <-listener.ctx.Done(): // Kill or ctrl-C
				errch <- context.Canceled
				return
			default:
			}

			// Peek for new messages
			n, _, err := unix.Recvfrom(listener.fd, buffer, unix.MSG_PEEK)
			if err != nil {
				errch <- fmt.Errorf(`could not peek: %v`, err)
				continue
			}

			if n == 0 {
				// Data length is zero, so go back to peeking new messages
				continue
			}

			if n < len(buffer) {
				// We have new message(s), break and handle them
				break
			}

			// Make buffer larger if needed
			buffer = make([]byte, len(buffer)*2)
		} // /end loop for peeking new messages

		// Read out all available messages
		n, fromtmp, err := unix.Recvfrom(listener.fd, buffer, 0)
		if err != nil {
			errch <- fmt.Errorf(`could not read messages: %v`, err)
			continue
		}

		if n == 0 {
			// Data length is zero, so go back to peeking new messages
			continue
		}

		from, ok := fromtmp.(*unix.SockaddrNetlink)
		if !ok {
			// Wrong type
			errch <- fmt.Errorf(`not SockaddrNetlink??`)
			continue
		}

		if from.Family != unix.AF_ROUTE {
			// Wrong family type
			errch <- fmt.Errorf(`not route type (AF_ROUTE)`)
			continue
		}

		msgs, err := syscall.ParseNetlinkMessage(buffer[:n])
		if err != nil {
			errch <- fmt.Errorf(`could not parse NetLink message: %w`, err)
			continue
		}

		for _, msg := range msgs {
			// io.Reader for marshalling data to struct(s)
			rdr := bytes.NewReader(msg.Data)
			che, err := parser.Parse(rdr)
			if err != nil {
				errch <- fmt.Errorf(`could not parse: %w`, err)
				continue
			}

			listener.onEventFunc(che)
		} // /for (messages)
	} // /for (main)
}

func (listener *Listener) Close() error {
	return unix.Close(listener.fd)
}
