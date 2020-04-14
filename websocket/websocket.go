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
		WsUrl            string
		Header           http.Header
		Subs             []string
		IsDump           bool
		IsAutoReconnect  bool
		readDeadLineTime time.Duration
	}

	WsClient struct {
		ctx    context.Context
		config *WsConfig
		conn   *websocket.Conn
		dialer *websocket.Dialer

		MessageFunc     func(msg []byte) error
		UncompressFunc  func(msg []byte) ([]byte, error)
		SystemErrorFunc func(err error)
	}
)

func NewBuilder() *WsBuilder {
	return &WsBuilder{
		config: &WsConfig{
			IsDump:          false,
			IsAutoReconnect: false,
		},
		dialer: websocket.DefaultDialer,
	}
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

func (b *WsBuilder) Build(ctx context.Context) *WsClient {

	client := &WsClient{
		ctx:    ctx,
		config: b.config,
		conn:   nil,
		dialer: b.dialer,

		MessageFunc:     DefaultMessageFunc,
		UncompressFunc:  DefaultUncompressFunc,
		SystemErrorFunc: SystemErrorFunc,
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
	logger.Error("", err)
}

func DefaultMessageFunc(msg []byte) error {
	logger.Info("[ws] receive message", string(msg))
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
			return err
		}
	}

	for retry := 1; retry <= 100; retry++ {
		time.Sleep(time.Duration(retry*10) * time.Second)
		err = w.Connect()
		if err != nil {
			w.SystemErrorFunc(errors.Wrap(err, fmt.Sprintf("[ws] websocket reconnect fail, retry[%d]", retry)))
			continue
		} else {
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
}

func (w *WsClient) ReceiveMessage() {

	var err error
	var msg []byte
	var messageType int
	for {
		messageType, msg, err = w.conn.ReadMessage()
		if err != nil {
			w.SystemErrorFunc(err)
			err := w.Reconnect()
			if err != nil {
				// Add Quit Handler
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
			logger.Info("[ws] abnormal close message", string(msg))
			w.Close()
			return
		case websocket.CloseMessage:
			logger.Info("[ws] close message", string(msg))
			w.Close()
			return
		case websocket.CloseGoingAway:
			logger.Info("[ws] goaway message", string(msg))
			w.Close()
			return
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

//// TODO 需要修改
//// Logger Interface , Default tech-botao logger
//func (w *WsClient) OnError(err error) {
//	_, _ = pp.Println(err)
//}
