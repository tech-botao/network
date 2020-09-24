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
		Subs([]string{`{"id": "id1", "sub": "market.btcusdt.kline.1min"}`}).
		Dump().
		AutoReconnect().
		ReadDeadLineTime(5*time.Second).
		ReconnectCount(1).
		ReconnectInterval(time.Millisecond*500).
		Dialer(websocket.DefaultDialer).
		Build(ctx, cancel)

	client.MessageFunc = func(msg []byte) error {
		if strings.Contains(string(msg), "ping") {
			pong := strings.Replace(string(msg), "ping", "pong", 1)
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

	client.GetConn().SetPingHandler(func(appData string) error {
		logger.Info("ping message", appData)
		return nil
	})
	client.GetConn().SetPongHandler(func(appData string) error {
		logger.Info("pong message", appData)
		return nil
	})

	client.GetConn().SetCloseHandler(func(code int, text string) error {
		logger.Info("close message, text:"+text, code)
		return nil
	})

	go client.ReceiveMessage()

	go func() {
		time.Sleep(100 * time.Second)
		cancel()
	}()

	select {
	case <-ctx.Done():
		logger.Info("exit timeout", time.Now().Format("2006-01-02 15:04:05"))
		return
	}

	// output:

}
