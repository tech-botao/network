package rest

import (
	"encoding/json"
	"fmt"
	"github.com/tech-botao/logger"
	"net/http"
	"net/http/httputil"
	"time"
)

type (
	PrepareHandler func(*http.Request)
	SuccessHandler func(*http.Response) error
	ErrorHandler   func(*http.Response, error)
	FinishHandler  func(*http.Response, time.Time)
)

type HttpClient struct {
	Client    *http.Client   // 可配置的HttpClient
	OnPrepare PrepareHandler // 访问前需要处理的地方
	OnSuccess SuccessHandler // 访问成功时接下来的处理
	OnError   ErrorHandler   // 当报错的时候的处理
	OnFinish  FinishHandler  // 访问结束时的处理
	IsDump    bool           // 是否Dump访问过程
}

// defaultPrepareHandler 那个请求都要执行的操作 [override]
func defaultPrepareHandler(req *http.Request) {
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	return
}

// defaultFinishHandler 默认访问时， 记录访问时间 [override]
func defaultFinishHandler(r *http.Response, start time.Time) {
	if r == nil {
		logger.Debug(fmt.Sprintf("[http] time [%.3f s], not response", time.Now().Sub(start).Seconds()), nil)
		return
	}
	logger.Debug(fmt.Sprintf("[http] time [%.3f s], response :", time.Now().Sub(start).Seconds()), ResponseDump(r))
}

// 默认成功的时候，Dump相关信息 [override]
func defaultSuccessHandler(response *http.Response) error {
	logger.Info(fmt.Sprintf("[http] succcess, content :"), ResponseDump(response))
	return nil
}

// 报错的时候 [override]
func defaultErrorHandler(response *http.Response, err error) {
	logger.Error(fmt.Sprintf("[http] failure, content : %s", ResponseDump(response)), err)
}

// NewHttpClient 创建默认对象
func NewHttpClient(c *http.Client) *HttpClient {
	if c == nil {
		c = http.DefaultClient
	}
	client := &HttpClient{
		Client:    c,
		OnPrepare: defaultPrepareHandler,
		OnSuccess: defaultSuccessHandler,
		OnError:   defaultErrorHandler,
		OnFinish:  defaultFinishHandler,
		IsDump:    false,
	}
	return client
}

// Dump 记录详细信息
func (h *HttpClient) Dump() *HttpClient {
	h.IsDump = true
	return h
}

// Result 直接取得response构造
func (h *HttpClient) Result(req *http.Request, v interface{}) error {
	return h.Do(req, func(response *http.Response) error {
		tmp := response.Body
		return json.NewDecoder(tmp).Decode(&v)
	})
}

// Do 执行一个response, 成功的时候执行next函数
func (h *HttpClient) Do(req *http.Request, next SuccessHandler) error {
	start := time.Now()
	if h.OnPrepare != nil {
		h.OnPrepare(req)
	}
	if h.IsDump && req != nil {
		logger.Info("[http] request", RequestDump(req))
	}
	response, err := h.Client.Do(req)
	defer func() {
		if h.OnFinish != nil {
			h.OnFinish(response, start)
		}
	}()
	if err != nil {
		h.OnError(response, err)
		return err
	}
	if response.StatusCode != 200 {
		h.OnError(response, fmt.Errorf("[http] response code is, %d", response.StatusCode))
		return fmt.Errorf("[http] response status is not 2xx, code = %d", response.StatusCode)
	}
	if next == nil {
		err = h.OnSuccess(response)
	} else {
		err = next(response)
	}
	return err
}

// ResponseDump 打印响应
func ResponseDump(response *http.Response) string {
	if response != nil {
		d, _ := httputil.DumpResponse(response, true)
		return string(d)
	}
	return ""
}

// RequestDump 打印请求
func RequestDump(req *http.Request) string {
	if req != nil {
		d, _ := httputil.DumpRequest(req, true)
		return string(d)
	}
	return ""
}

