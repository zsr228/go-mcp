package client

import (
	"context"
	"fmt"
	"sync/atomic"

	"go-mcp/pkg"
	"go-mcp/protocol"
	"go-mcp/transport"

	cmap "github.com/orcaman/concurrent-map/v2"
)

type Option func(*Client)

func WithToolsListChangedNotifyHandler(handler func(ctx context.Context, request *protocol.ToolListChangedNotification) error) Option {
	return func(s *Client) {
		s.notifyHandlerWithToolsListChanged = handler
	}
}

func WithPromptListChangedNotifyHandler(handler func(ctx context.Context, request *protocol.PromptListChangedNotification) error) Option {
	return func(s *Client) {
		s.notifyHandlerWithPromptListChanged = handler
	}
}

func WithResourceListChangedNotifyHandler(handler func(ctx context.Context, request *protocol.ResourceListChangedNotification) error) Option {
	return func(s *Client) {
		s.notifyHandlerWithResourceListChanged = handler
	}
}

func WithResourcesUpdatedNotifyHandler(handler func(ctx context.Context, request *protocol.ResourceUpdatedNotification) error) Option {
	return func(s *Client) {
		s.notifyHandlerWithResourcesUpdated = handler
	}
}

func WithClientInfo(info protocol.Implementation) Option {
	return func(s *Client) {
		s.clientInfo = &info
	}
}

func WithLogger(logger pkg.Logger) Option {
	return func(s *Client) {
		s.logger = logger
	}
}

type Client struct {
	transport transport.ClientTransport

	reqID2respChan cmap.ConcurrentMap[string, chan *protocol.JSONRPCResponse]

	notifyHandlerWithToolsListChanged    func(ctx context.Context, request *protocol.ToolListChangedNotification) error
	notifyHandlerWithPromptListChanged   func(ctx context.Context, request *protocol.PromptListChangedNotification) error
	notifyHandlerWithResourceListChanged func(ctx context.Context, request *protocol.ResourceListChangedNotification) error
	notifyHandlerWithResourcesUpdated    func(ctx context.Context, request *protocol.ResourceUpdatedNotification) error

	requestID atomic.Int64

	ready atomic.Bool

	clientInfo         *protocol.Implementation
	clientCapabilities *protocol.ClientCapabilities

	serverCapabilities *protocol.ServerCapabilities
	serverInfo         *protocol.Implementation
	serverInstructions string

	logger pkg.Logger
}

func NewClient(t transport.ClientTransport, opts ...Option) (*Client, error) {
	client := &Client{
		transport:          t,
		logger:             pkg.DefaultLogger,
		reqID2respChan:     cmap.New[chan *protocol.JSONRPCResponse](),
		clientInfo:         &protocol.Implementation{},
		clientCapabilities: &protocol.ClientCapabilities{},
	}
	t.SetReceiver(client)

	for _, opt := range opts {
		opt(client)
	}

	if err := client.transport.Start(); err != nil {
		return nil, fmt.Errorf("init mcp client transpor start fail: %w", err)
	}

	if _, err := client.initialization(context.Background(), protocol.NewInitializeRequest(*client.clientInfo, *client.clientCapabilities)); err != nil {
		return nil, err
	}

	return client, nil
}

func (client *Client) GetServerCapabilities() protocol.ServerCapabilities {
	return *client.serverCapabilities
}

func (client *Client) GetServerInfo() protocol.Implementation {
	return *client.serverInfo
}

func (client *Client) GetServerInstructions() string {
	return client.serverInstructions
}

func (client *Client) Close() error {
	if err := client.transport.Close(); err != nil {
		return err
	}
	return nil
}
