package rest

import (
	"bytes"
	"fmt"
	"github.com/tech-botao/logger"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func _newClient() *HttpClient {
	return NewHttpClient(http.DefaultClient)
}

func _newRequest(method, url string, data io.Reader) *http.Request {
	req, _ := http.NewRequest(method, url, data)
	return req
}

var dumpHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
	_, _ = fmt.Fprintf(w, RequestDump(r))
})

var jsonHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
	b , err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Panic("[jsonHandler], ", err)
	}
	if len(b) == 0 {
		_, _ = fmt.Fprintf(w, `{}`)

	} else {
		_, _ = fmt.Fprintf(w, string(b))
	}
})

func TestNormal(t *testing.T) {
	ts := httptest.NewServer(dumpHandler)
	defer ts.Close()

	r, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("[httptest] http.Get(), %v", err)
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("[httptest] ioutil.ReadAll(), %v", err)
	}

	fmt.Printf("%v\n", string(data))
}

func TestHttpClient_Do(t *testing.T) {
	ts := httptest.NewServer(dumpHandler)
	defer ts.Close()

	type args struct {
		req  *http.Request
		next SuccessHandler
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"GET", args{ req:  _newRequest("GET", ts.URL, nil), next: nil, }, false},
		{"POST", args{ req:  _newRequest("POST", ts.URL, bytes.NewReader([]byte(`{"a":1,"b":"c"}`))), next: nil, }, false},
		{"DELETE", args{ req:  _newRequest("DELETE", ts.URL, bytes.NewReader([]byte(`{"a":1,"b":"c"}`))), next: nil, }, false},
		{"PUT", args{ req:  _newRequest("PUT", ts.URL, bytes.NewReader([]byte(`{"a":1,"b":"c"}`))), next: nil, }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := _newClient().Dump().Do(tt.args.req, tt.args.next); (err != nil) != tt.wantErr {
				t.Errorf("Do() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHttpClient_Result(t *testing.T) {

	ts := httptest.NewServer(jsonHandler)
	defer ts.Close()

	var data interface{}
	type args struct {
		req *http.Request
		v   interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"GET", args{ req:  _newRequest("GET", ts.URL, nil), v: &data}, false},
		{"POST", args{ req:  _newRequest("POST", ts.URL, bytes.NewReader([]byte(`{"a":1,"b":"c"}`))), v: &data, }, false},
		{"DELETE", args{ req:  _newRequest("DELETE", ts.URL, bytes.NewReader([]byte(`{"a":1,"b":"c"}`))), v: &data, }, false},
		{"PUT", args{ req:  _newRequest("PUT", ts.URL, bytes.NewReader([]byte(`{"a":1,"b":"c"}`))), v: &data, }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := _newClient().Result(tt.args.req, tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("Result() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
		fmt.Printf("%v\n", tt.args.v)
	}
}

func TestHttpClient_Dump(t *testing.T) {
	t.Skip("defaultSuccessHandler is simple, dont test")
}

func TestNewHttpClient(t *testing.T) {
	t.Skip("NewHttpClient is include Do(), dont test")
}

func TestRequestDump(t *testing.T) {
	t.Skip("RequestDump is simple, dont test")
}

func TestResponseDump(t *testing.T) {
	t.Skip("RequestDump is simple, dont test")
}

func Test_defaultErrorHandler(t *testing.T) {
	t.Skip("defaultErrorHandler is simple, dont test")
}

func Test_defaultFinishHandler(t *testing.T) {
	t.Skip("defaultFinishHandler is simple, dont test")
}

func Test_defaultPrepareHandler(t *testing.T) {
	t.Skip("defaultPrepareHandler is simple, dont test")
}

func Test_defaultSuccessHandler(t *testing.T) {
	t.Skip("defaultSuccessHandler is simple, dont test")
}