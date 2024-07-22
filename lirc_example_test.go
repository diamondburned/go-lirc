package lirc_test

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"libdb.so/go-lirc"
)

func Example() {
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

func Example_sendCommand() {
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

	for _, command := range []lirc.Command{
		lirc.List{RemoteControl: "DenonTuner"},
		lirc.SendOnce{RemoteControl: "DenonTuner", ButtonName: "PROG-SCAN"},
	} {
		resp, err := conn.SendCommand(ctx, command)
		if err != nil {
			slog.Error(
				"cannot send version command",
				"err", err)
			return
		}

		slog.Info(
			"lirc reply",
			"command", command.EncodeCommand()[0],
			"reply", resp.Data)
	}

	// WaitGroup omitted for brevity.
}

func ExampleRouteEvents() {
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

	go lirc.RouteEvents(ctx, conn.Events, lirc.RemoteHandlers{
		"*": lirc.ButtonHandlers{
			"KEY_POWER": func(lirc.ButtonPress) { slog.Info("power button pressed") },
			"KEY_TV":    func(lirc.ButtonPress) { slog.Info("tv button pressed") },
			"*":         func(lirc.ButtonPress) { slog.Info("unknown button pressed") },
		},
	})

	<-ctx.Done()

	// WaitGroup omitted for brevity.
}
