package transport

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
)

const stdioSessionID = "stdio"

type StdioServerTransportOption func(*stdioServerTransport)

func WithStdioServerOptionLogger(log pkg.Logger) StdioServerTransportOption {
	return func(t *stdioServerTransport) {
		t.logger = log
	}
}

type stdioServerTransport struct {
	receiver ServerReceiver
	reader   io.ReadCloser
	writer   io.Writer

	logger pkg.Logger

	cancel          context.CancelFunc
	receiveShutDone chan struct{}
}

func NewStdioServerTransport(opts ...StdioServerTransportOption) ServerTransport {
	t := &stdioServerTransport{
		reader: os.Stdin,
		writer: os.Stdout,
		logger: pkg.DefaultLogger,

		receiveShutDone: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *stdioServerTransport) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel

	t.receive(ctx)

	close(t.receiveShutDone)
	return nil
}

func (t *stdioServerTransport) Send(ctx context.Context, sessionID string, msg Message) error {
	if _, err := t.writer.Write(append(msg, mcpMessageDelimiter)); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}
	return nil
}

func (t *stdioServerTransport) SetReceiver(receiver ServerReceiver) {
	t.receiver = receiver
}

func (t *stdioServerTransport) Shutdown(userCtx context.Context, serverCtx context.Context) error {
	// After calling `close` on `os.Stdin` in Go, it does not support interrupting an ongoing read operation; therefore, `stdioServerTransport` does not support `Shutdown`.
	return fmt.Errorf("stdioServerTransport not support shutdown, please listen Run()")
}

func (t *stdioServerTransport) receive(ctx context.Context) {
	s := bufio.NewScanner(t.reader)

	for s.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			if err := t.receiver.Receive(ctx, stdioSessionID, s.Bytes()); err != nil {
				t.logger.Errorf("receiver failed: %v", err)
				return
			}
		}
	}

	if err := s.Err(); err != nil {
		if !errors.Is(err, io.ErrClosedPipe) { // This error occurs during unit tests, suppressing it here
			t.logger.Errorf("server server unexpected error reading input: %v", err)
		}
		return
	}
}
