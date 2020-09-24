package websocket

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/tech-botao/logger"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"time"
)

type (
	WsBuilder struct {
		config *WsConfig
		dialer *websocket.Dialer
	}

	WsConfig struct {
		WsUrl             string
		Header            http.Header
		Subs              []string
		IsDump            bool
		IsAutoReconnect   bool
		readDeadLineTime  time.Duration
		reconnectCount    int
		reconnectInterval time.Duration
	}

	WsClient struct {
		ctx    context.Context
		config *WsConfig
		conn   *websocket.Conn
		dialer *websocket.Dialer
		cancel context.CancelFunc

		MessageFunc        func(msg []byte) error
		UncompressFunc     func(msg []byte) ([]byte, error)
		SystemErrorFunc    func(err error)
		AfterConnectedFunc func() error
	}
)

// GetConn 返回Conn的
func (w *WsClient) GetConn() *websocket.Conn {
	return w.conn
}

func NewBuilder() *WsBuilder {
	return &WsBuilder{
		config: &WsConfig{
			IsDump:            false,
			IsAutoReconnect:   false,
			reconnectInterval: time.Second,
			reconnectCount:    100,
		},
		dialer: websocket.DefaultDialer,
	}
}

func (b *WsBuilder) ReconnectCount(count int) *WsBuilder {
	b.config.reconnectCount = count
	return b
}

func (b *WsBuilder) ReconnectInterval(interval time.Duration) *WsBuilder {
	b.config.reconnectInterval = interval
	return b
}

func (b *WsBuilder) URL(url string) *WsBuilder {
	b.config.WsUrl = url
	return b
}

func (b *WsBuilder) Header(header http.Header) *WsBuilder {
	b.config.Header = header
	return b
}

func (b *WsBuilder) Subs(subs []string) *WsBuilder {
	b.config.Subs = subs
	return b
}

func (b *WsBuilder) AutoReconnect() *WsBuilder {
	b.config.IsAutoReconnect = true
	return b
}

func (b *WsBuilder) Dump() *WsBuilder {
	b.config.IsDump = true
	return b
}

func (b *WsBuilder) Dialer(d *websocket.Dialer) *WsBuilder {
	b.dialer = d
	return b
}

func (b *WsBuilder) ReadDeadLineTime(t time.Duration) *WsBuilder {
	b.config.readDeadLineTime = t
	return b
}

func (b *WsBuilder) Build(ctx context.Context, cancel context.CancelFunc) *WsClient {

	var client = &WsClient{
		ctx:    ctx,
		config: b.config,
		conn:   nil,
		dialer: b.dialer,
		cancel: cancel,

		MessageFunc:        DefaultMessageFunc,
		UncompressFunc:     DefaultUncompressFunc,
		SystemErrorFunc:    SystemErrorFunc,
		AfterConnectedFunc: DefaultAfterConnected,
	}
	return client
}

func DefaultUncompressFunc(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

func DumpResponse(resp *http.Response, body bool) {

	if resp == nil {
		return
	}

	d, _ := httputil.DumpResponse(resp, body)
	logger.Info("[ws] response:", string(d))
}

func SystemErrorFunc(err error) {
	logger.Error("[ws] system error", err.Error())
}

func DefaultMessageFunc(msg []byte) error {
	logger.Info("[ws] receive message", string(msg))
	return nil
}

func DefaultAfterConnected() error {
	logger.Info("[ws] connect success.", time.Now().Format("2006-01-02 15:04:05"))
	return nil
}

func (w *WsClient) dump(resp *http.Response, body bool) {
	if w.config.IsDump {
		DumpResponse(resp, body)
	}
}

func (w *WsClient) Connect() error {
	conn, resp, err := w.dialer.Dial(w.config.WsUrl, w.config.Header)
	defer w.dump(resp, true)
	if err != nil {
		return err
	}
	w.conn = conn

	err = w.AfterConnectedFunc()
	if err != nil {
		return err
	}

	if len(w.config.Subs) > 0 {
		for _, s := range w.config.Subs {
			err = w.conn.WriteMessage(websocket.TextMessage, []byte(s))
			if err != nil {
				w.SystemErrorFunc(err)
			}
		}
	}
	return err
}

func (w *WsClient) Reconnect() error {
	var err error
	if w.conn != nil {
		err = w.conn.Close()
		if err != nil {
			w.SystemErrorFunc(errors.Wrapf(err, "[ws] [%s] close websocket error", w.config.WsUrl))
			w.conn = nil
			//return err
		}
	}

	for retry := 1; retry <= w.config.reconnectCount; retry++ {
		time.Sleep(w.config.reconnectInterval)
		err = w.Connect()
		if err != nil {
			w.SystemErrorFunc(errors.Wrap(err, fmt.Sprintf("[ws] websocket reconnect fail, retry[%d]", retry)))
			continue
		} else {
			logger.Info("[ws] retry", retry)
			break
		}
	}

	return err
}

func (w *WsClient) Close() {
	err := w.conn.Close()
	if err != nil {
		w.SystemErrorFunc(errors.Wrapf(err, "[ws] [%s] close websocket error", w.config.WsUrl))
	} else {
		logger.Info("[ws] close websocket success.", nil)
	}
	time.Sleep(time.Second)
	w.cancel()
}

func (w *WsClient) ReceiveMessage() {
	var err error
	var msg []byte
	var messageType int
	defer w.Close()
	for {
		messageType, msg, err = w.conn.ReadMessage()
		if err != nil {
			w.SystemErrorFunc(err)
			err := w.Reconnect()
			if err != nil {
				w.SystemErrorFunc(errors.Wrap(err, "[ws] quit message loop."))
				return
			}
		}

		// 收到信息后， 延长时间
		if w.config.readDeadLineTime > 0 {
			err = w.conn.SetReadDeadline(time.Now().Add(w.config.readDeadLineTime))
			if err != nil {
				logger.Warn("set readDeadLine error", err)
			}
		}

		switch messageType {
		case websocket.TextMessage:
			err = w.MessageFunc(msg)
			if err != nil {
				w.SystemErrorFunc(errors.Wrap(err, "[ws] message handler error."))
			}
		case websocket.BinaryMessage:
			msg, err := w.UncompressFunc(msg)
			if err != nil {
				w.SystemErrorFunc(errors.Wrap(err, "[ws] uncompress handler error."))
			} else {
				err = w.MessageFunc(msg)
				if err != nil {
					w.SystemErrorFunc(errors.Wrap(err, "[ws] uncompress message handler error."))
				}
			}
		case websocket.CloseAbnormalClosure:
			w.SystemErrorFunc(errors.Wrap(fmt.Errorf("%s", string(msg)), "[ws] abnormal close message"))
		case websocket.CloseMessage:
			w.SystemErrorFunc(errors.Wrap(fmt.Errorf("%s", string(msg)), "[ws] close message"))
		case websocket.CloseGoingAway:
			w.SystemErrorFunc(errors.Wrap(fmt.Errorf("%s", string(msg)), "[ws] goaway message"))
		case websocket.PingMessage:
			logger.Info("[ws] receive ping", string(msg))
		case websocket.PongMessage:
			logger.Info("[ws] receive pong", string(msg))
		default:
			logger.Error(fmt.Sprintf("[ws][%s] error websocket messageType = %d", w.config.WsUrl, messageType), msg)
		}
	}
}

func (w *WsClient) WriteMessage(messageType int, msg []byte) error {
	return w.conn.WriteMessage(messageType, msg)
}

func (w *WsClient) WriteControl(messageType int, msg []byte, deadline time.Time) error {
	return w.conn.WriteControl(messageType, msg, deadline)
}
