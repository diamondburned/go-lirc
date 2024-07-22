# lirc

[![Go Reference](https://pkg.go.dev/badge/libdb.so/lirc.svg)](https://pkg.go.dev/libdb.so/lirc)

Package lirc provides a Go client for the Linux Infrared Remote Control (LIRC)
daemon.

## Importing

```go
import "libdb.so/lirc"
```

## Example

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"libdb.so/lirc"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	conn := lirc.NewUnix("/run/lirc/lircd")

	go func() {
		if err := conn.Start(ctx, slog.Default()); err != nil {
			slog.Error(
				"lirc connection failed",
				"err", err)
		}
		cancel()
	}()

	resp, err := conn.SendCommand(ctx, lirc.Version{})
	if err != nil {
		slog.Error(
			"cannot send version command",
			"err", err)
		return
	}

	slog.Info(
		"lirc version",
		"version", resp.Data[0])

	// WaitGroup omitted for brevity.
}
```

See the reference documentation for more examples.
