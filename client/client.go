package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Request struct {
	BaseUrl          string
	Headers          map[string]string
	HttpClient       *http.Client
	HttpErrorHandler HttpErrorHandler
}

type HttpErrorHandler func(res *http.Response, uri string) error

func (r *Request) SetTimeout(seconds time.Duration) {
	r.HttpClient.Timeout = time.Second * seconds
}

func InitClient(baseUrl string, errorHandler HttpErrorHandler) Request {
	if errorHandler == nil {
		errorHandler = DefaultErrorHandler
	}
	return Request{
		Headers:          make(map[string]string),
		HttpClient:       DefaultClient,
		HttpErrorHandler: errorHandler,
		BaseUrl:          baseUrl,
	}
}

func InitJSONClient(baseUrl string, errorHandler HttpErrorHandler) Request {
	client := InitClient(baseUrl, errorHandler)
	client.Headers = map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	return client
}

var DefaultClient = &http.Client{
	Timeout: time.Second * 15,
}

var DefaultErrorHandler = func(res *http.Response, uri string) error {
	return nil
}

func (r *Request) AddHeader(key string, value string) {
	r.Headers[key] = value
}

func (r *Request) GetWithContext(result interface{}, path string, query url.Values, ctx context.Context) error {
	var queryStr = ""
	if query != nil {
		queryStr = query.Encode()
	}
	uri := strings.Join([]string{r.GetBase(path), queryStr}, "?")
	return r.Execute("GET", uri, nil, result, ctx)
}

func (r *Request) Get(result interface{}, path string, query url.Values) error {
	var queryStr = ""
	if query != nil {
		queryStr = query.Encode()
	}
	uri := strings.Join([]string{r.GetBase(path), queryStr}, "?")
	return r.Execute("GET", uri, nil, result, context.Background())
}

func (r *Request) Post(result interface{}, path string, body interface{}) error {
	buf, err := GetBody(body)
	if err != nil {
		return err
	}
	uri := r.GetBase(path)
	return r.Execute("POST", uri, buf, result, context.Background())
}

func (r *Request) PostWithContext(result interface{}, path string, body interface{}, ctx context.Context) error {
	buf, err := GetBody(body)
	if err != nil {
		return err
	}
	uri := r.GetBase(path)
	return r.Execute("POST", uri, buf, result, ctx)
}

func (r *Request) Execute(method string, url string, body io.Reader, result interface{}, ctx context.Context) error {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}

	for key, value := range r.Headers {
		req.Header.Set(key, value)
	}

	c := r.HttpClient

	res, err := c.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}

	err = r.HttpErrorHandler(res, url)
	if err != nil {
		return err
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("failed request status: %s for url: %s", strconv.Itoa(res.StatusCode), res.Request.RequestURI)
	}

	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, result)
	if err != nil {
		return err
	}
	return err
}

func (r *Request) GetBase(path string) string {
	baseUrl := strings.TrimRight(r.BaseUrl, "/")
	if path == "" {
		return baseUrl
	}
	path = strings.TrimLeft(path, "/")
	return fmt.Sprintf("%s/%s", baseUrl, path)
}

func GetBody(body interface{}) (buf io.ReadWriter, err error) {
	if body != nil {
		buf = new(bytes.Buffer)
		err = json.NewEncoder(buf).Encode(body)
	}
	return
}
