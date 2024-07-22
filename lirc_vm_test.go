package lirc_test

import (
	"context"
	"os"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/neilotoole/slogt"
	"libdb.so/lirc"
)

func TestVersion(t *testing.T) {
	conn, ctx := startConnection(t)

	reply, err := conn.SendCommand(ctx, lirc.Version{})
	assert.NoError(t, err, "send version command")
	assert.Equal(t, []string{"0.10.2"}, reply.Data, "version data")
}

func startConnection(t *testing.T) (*lirc.Connection, context.Context) {
	testUnixAddress := os.Getenv("LIRC_TEST_UNIX_ADDRESS")
	if testUnixAddress == "" {
		t.Skip("LIRC_TEST_UNIX_ADDRESS is not set")
	}

	conn := lirc.NewUnix(testUnixAddress)
	ctx, cancel := context.WithCancelCause(context.Background())

	errCh := make(chan error, 1)
	go func() {
		logger := slogt.New(t).With("module", "lirc")
		err := conn.Start(ctx, logger)
		cancel(err)
		errCh <- err
	}()

	t.Cleanup(func() {
		cancel(nil)
		if err := <-errCh; err != nil && err != context.Canceled {
			assert.NoError(t, <-errCh, "lirc connection failed")
		}
	})

	return conn, ctx
}
