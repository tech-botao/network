package websocket

import (
	"context"
	"github.com/gorilla/websocket"
	"github.com/k0kubun/pp"
	"github.com/tech-botao/logger"
	"strings"
	"time"
)

func ExampleHbg() {

	ctx, cancel := context.WithCancel(context.Background())
	client := NewBuilder().URL("wss://api.huobi.pro/ws").
		Subs([]string{`{"id": "id1", "sub": "market.btcusdt.kline.1min"}`,}).
		Dump().
		AutoReconnect().
		Dialer(websocket.DefaultDialer).
		Build(ctx)

	client.MessageFunc = func(msg []byte) error {
		if strings.Contains(string(msg), "ping") {
			pong := strings.Replace(string(msg),"ping", "pong", 1)
			logger.Info("pong", pong)
			return client.WriteMessage(websocket.TextMessage, []byte(pong))
		}
		logger.Info("[ws] receive message", string(msg))
		return nil
	}

	err := client.Connect()
	if err != nil {
		pp.Println(err)
	}

	go client.ReceiveMessage()

	go func() {
		time.Sleep(100 * time.Second)
		cancel()
	}()

	select {
	case <-ctx.Done():
		client.Close()
	}

	// output:

}
