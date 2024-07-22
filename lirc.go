// Package lirc provides a Go client for the Linux Infrared Remote Control
// (LIRC) daemon.
package lirc

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Connection is a connection to lircd.
type Connection struct {
	// Events is a channel that will receive ButtonPress events.
	// These events are received asynchronously for as long as [Start] is
	// running. This channel is never closed.
	Events chan ButtonPress

	send   chan Command
	reply  chan CommandReply
	dialer func(context.Context) (net.Conn, error)
}

// NewUnix creates a new lirc connection that connects to lircd using a Unix
// socket.
// Connection will not be established; you must call Start to connect to lircd.
func NewUnix(path string) *Connection {
	return newRouter(func(ctx context.Context) (net.Conn, error) {
		return net.DefaultResolver.Dial(ctx, "unix", path)
	})
}

// NewTCP creates a new lirc connection that connects to lircd using a TCP
// socket.
// Connection will not be established; you must call Start to connect to lircd.
func NewTCP(host string) *Connection {
	return newRouter(func(ctx context.Context) (net.Conn, error) {
		return net.DefaultResolver.Dial(ctx, "tcp", host)
	})
}

func newRouter(dialer func(ctx context.Context) (net.Conn, error)) *Connection {
	return &Connection{
		Events: make(chan ButtonPress),
		send:   make(chan Command),
		reply:  make(chan CommandReply),
		dialer: dialer,
	}
}

// SendCommand sends a command to lirc daemon.
func (l *Connection) SendCommand(ctx context.Context, command Command) (CommandReply, error) {
	select {
	case <-ctx.Done():
		return CommandReply{}, ctx.Err()
	case l.send <- command:
		// safe to continue
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		return CommandReply{}, ctx.Err()
	case reply := <-l.reply:
		if reply.Command != command.EncodeCommand()[0] {
			return reply, fmt.Errorf("unexpected reply command: %q", reply.Command)
		}
		if !reply.Success {
			return reply, ErrUnsuccessfulCommand
		}
		return reply, nil
	}
}

// RepeatButton tells lircd to keep sending the given button until the returned
// callback is called.
func (l *Connection) RepeatButton(ctx context.Context, remote, button string) (stop func(), err error) {
	if _, err := l.SendCommand(ctx, SendStart{remote, button}); err != nil {
		return nil, err
	}

	return func() {
		l.SendCommand(ctx, SendStop{remote, button})
	}, nil
}

type connectionState uint

const (
	stateReceive connectionState = iota
	stateReply
	stateMessage
	stateStatus
	stateDataStart
	stateDataLength
	stateData
	stateDataEnd
)

// Start starts the lirc connection. It blocks until the connection is closed or
// ctx is done.
func (r *Connection) Start(ctx context.Context, logger *slog.Logger) error {
	conn, err := r.dialer(ctx)
	if err != nil {
		return fmt.Errorf("cannot dial lircd connection: %w", err)
	}

	logger = logger.With("connection", conn.RemoteAddr().String())

	repliesCh := make(chan CommandReply)
	sendingCh := r.send

	reader := newLircReader(logger, r.Events, repliesCh)

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancelCause(ctx)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel(nil)

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			reader.read(ctx, line)
		}

		if err := scanner.Err(); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Error(
				"error reading from lircd socket",
				"err", err)
			cancel(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel(nil)

		var cmd Command
		for {
			select {
			case <-ctx.Done():
				return

			case cmd = <-sendingCh:
				raw := strings.Join(cmd.EncodeCommand(), " ") + "\n"
				if _, err := io.WriteString(conn, raw); err != nil {
					logger.Error(
						"error writing to lircd socket",
						"err", err)
					cancel(err)
					return
				}

				// Prevent the user from sending any other commands until we've
				// received the reply for this one.
				sendingCh = nil

			case reply := <-repliesCh:
				if reply.Command == "SIGHUP" {
					logger.InfoContext(ctx, "lircd has been reloaded")
					continue
				}

				select {
				case <-ctx.Done():
					return
				case r.reply <- reply:
					// Reinstate the ability to send commands.
					sendingCh = r.send
				}
			}
		}
	}()

	<-ctx.Done()

	if err := conn.Close(); err != nil {
		return fmt.Errorf("error closing lircd connection: %w", err)
	}

	wg.Wait()
	return context.Cause(ctx)
}

type lircReader struct {
	state      connectionState
	reply      CommandReply
	dataCount  int
	dataLength int

	logger  *slog.Logger
	events  chan ButtonPress
	replies chan CommandReply
}

func newLircReader(logger *slog.Logger, events chan ButtonPress, replies chan CommandReply) *lircReader {
	return &lircReader{
		state:   stateReceive,
		logger:  logger,
		events:  events,
		replies: replies,
	}
}

func (r *lircReader) stateError(err string, attrs ...any) {
	r.logger.
		With("err", err).
		Error("lirc error", attrs...)
	r.state = stateReceive
}

func (r *lircReader) flushReply(ctx context.Context) {
	select {
	case <-ctx.Done():
		// no
	case r.replies <- r.reply:
	}
}

func (r *lircReader) read(ctx context.Context, line string) {
	switch r.state {
	case stateReceive:
		if line == "BEGIN" {
			r.state = stateReply

			r.reply = CommandReply{}
			r.dataCount = 0
			r.dataLength = 0

			return
		}

		w := strings.Split(line, " ")
		h := w[0]
		if len(h) < 16 {
			h = strings.Repeat("0", 16-len(h)) + h
		}

		c, err := hex.DecodeString(h)
		if err != nil {
			r.stateError(
				"lirc code not parseable as hex",
				"len", len(h))
			return
		}
		if len(c) != 8 {
			r.stateError(
				"lirc code has wrong length for 16-bit integer",
				"len", len(c))
			return
		}

		repeats, err := strconv.ParseUint(w[1], 10, 0)
		if err != nil {
			r.stateError(
				"lirc repeat count not parseable as decimal (invalid repeat count)",
				"repeat", w[1])
			return
		}

		event := ButtonPress{
			Code:              binary.LittleEndian.Uint16(c),
			RepeatCount:       uint(repeats),
			ButtonName:        w[2],
			RemoteControlName: w[3],
		}

		select {
		case <-ctx.Done():
			return
		case r.events <- event:
		}

	case stateReply:
		r.reply = CommandReply{
			Command: line,
			Success: true,
		}
		r.state = stateStatus

	case stateStatus:
		switch line {
		case "SUCCESS":
			r.state = stateDataStart
		case "ERROR":
			r.reply.Success = false
			r.state = stateDataStart
		case "END":
			r.state = stateReceive
			r.flushReply(ctx)
		default:
			r.stateError(
				"lirc reply message received has invalid status",
				"line", line)
			return
		}

	case stateDataStart:
		switch line {
		case "DATA":
			r.state = stateDataLength
		case "END":
			r.state = stateReceive
			r.flushReply(ctx)
		default:
			r.stateError(
				"lirc reply message received has invalid data start",
				"line", line)
			return
		}

	case stateDataLength:
		var err error
		r.dataLength, err = strconv.Atoi(line)
		if err != nil {
			r.stateError(
				"lirc reply message received has invalid data length",
				"line", line)
			return
		}

		r.state = stateData
		r.dataCount = 0
		r.reply.Data = make([]string, 0, r.dataLength)

	case stateData:
		r.reply.Data = append(r.reply.Data, line)
		r.dataCount++
		if r.dataCount > r.dataLength {
			r.state = stateDataEnd
		}

	case stateDataEnd:
		if line != "END" {
			r.stateError(
				"lirc reply message received has invalid data end, discarding reply",
				"line", line)
			return
		}

		r.flushReply(ctx)
		r.state = stateReceive
	}
}
