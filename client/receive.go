package client

import (
	"context"
	"fmt"

	"go-mcp/pkg"
	"go-mcp/protocol"

	"github.com/tidwall/gjson"
)

// 对来自客户端的 message(request、response、notification)进行接收处理
// 如果是 request、notification 路由到对应的handler处理，如果是 response 则传递给对应 reqID 的 chan

func (client *Client) Receive(ctx context.Context, msg []byte) error {
	defer pkg.Recover()

	if !gjson.GetBytes(msg, "id").Exists() {
		notify := &protocol.JSONRPCNotification{}
		if err := pkg.JsonUnmarshal(msg, &notify); err != nil {
			return err
		}
		go func() {
			defer pkg.Recover()

			if err := client.receiveNotify(ctx, notify); err != nil {
				// TODO: 打印日志
				return
			}
		}()
		return nil
	}

	// 判断 request和response
	if !gjson.GetBytes(msg, "method").Exists() {
		resp := &protocol.JSONRPCResponse{}
		if err := pkg.JsonUnmarshal(msg, &resp); err != nil {
			// 打印日志
			return err
		}
		go func() {
			defer pkg.Recover()

			if err := client.receiveResponse(ctx, resp); err != nil {
				// TODO: 打印日志
				return
			}
		}()
		return nil
	}

	req := &protocol.JSONRPCRequest{}
	if err := pkg.JsonUnmarshal(msg, &req); err != nil {
		// 打印日志
		return err
	}
	go func() {
		defer pkg.Recover()

		if err := client.receiveRequest(ctx, req); err != nil {
			// TODO: 打印日志
			return
		}
	}()
	return nil
}

func (client *Client) receiveRequest(ctx context.Context, request *protocol.JSONRPCRequest) error {
	if !request.IsValid() {
		// return protocol.NewJSONRPCErrorResponse(request.ID,)
	}

	// TODO：此处需要根据 request.Method 判断客户端是否声明此能力，如果未声明则报错返回。

	var (
		result protocol.ClientResponse
		err    error
	)

	switch request.Method {
	case protocol.Ping:
		result, err = client.handleRequestWithPing()
	case protocol.RootsList:
		result, err = client.handleRequestWithListRoots(ctx, request.RawParams)
	case protocol.SamplingCreateMessage:
		result, err = client.handleRequestWithCreateMessagesSampling(ctx, request.RawParams)
	default:
		err = fmt.Errorf("request method=%s not supoort", request.Method)
	}

	if err != nil {
		// TODO: 此处需要根据err的类型传入不同的错误码
		return client.sendMsgWithError(ctx, request.ID, protocol.INVALID_REQUEST, err.Error())
	}
	return client.sendMsgWithResponse(ctx, request.ID, result)
}

func (client *Client) receiveNotify(ctx context.Context, notify *protocol.JSONRPCNotification) error {
	switch notify.Method {
	case protocol.NotificationCancelled:
		return client.handleNotifyWithCancelled(ctx, notify.RawParams)
	case protocol.NotificationProgress:
		// TODO
	case protocol.NotificationToolsListChanged:
		// TODO
	case protocol.NotificationPromptsListChanged:
		// TODO
	case protocol.NotificationResourcesListChanged:
		// TODO
	case protocol.NotificationResourcesUpdated:
	// TODO
	case protocol.NotificationLogMessage:
		// TODO
	default:
		// TODO: return pkg.errors
		return nil
	}
	return nil
}

func (client *Client) receiveResponse(ctx context.Context, response *protocol.JSONRPCResponse) error {
	respChan, ok := client.reqID2respChan.Get(fmt.Sprint(response.ID))
	if !ok {
		return fmt.Errorf("%w: requestID=%+v", pkg.ErrLackResponseChan, response.ID)
	}

	select {
	case respChan <- response:
	default:
		return fmt.Errorf("duplicate response received: response=%+v", response)
	}
	return nil
}
