package http

import (
	"crypto/tls"
	"github.com/go-resty/resty/v2"
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
	"time"
)

type Interceptor = func(method string, url string, body interface{}, headers Headers) *Response
type Headers = map[string]string
type FormData = map[string]string

type Client interface {
	Get(url string, headers *Headers) (Response, error)
	PostForm(url string, formData FormData, headers *Headers) (Response, error)
	Post(url string, payload interface{}, headers *Headers) (Response, error)
	Delete(url string, payload interface{}, headers *Headers) (Response, error)
}

type Response struct {
	Status  int
	Body    []byte
	IsError bool
	Err     error
}

type DefaultHttpClient struct {
	client *resty.Client
}

var (
	httpInterceptor Interceptor
)


func Intercept(interceptor Interceptor) {
	httpInterceptor = interceptor
}

func NewHttpClient(debug bool) Client {
	client := resty.New()
	client.SetDebug(debug)
	// client.SetTLSClientConfig(&tls.Config{ RootCAs: roots })
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	// Set client timeout as per your need
	client.SetTimeout(30 * time.Second)
	return DefaultHttpClient{
		client: client,
	}
}

func (c Response) Decode(dest interface{}) error {
	return h.FromJson(c.Body, dest)
}

func (c *Response) WithJsonBody(body interface{}) *Response {
	data, _ := h.ToJson(body)
	c.Body = data
	return c
}
func NewHttpResponse(status int, body interface{}) *Response {
	data, _ := h.ToJson(body)
	return &Response{Status: status, Body: data}
}

func (c DefaultHttpClient) Get(url string, headers *Headers) (Response, error) {
	h := Headers{}
	if headers != nil {
		h = *headers
	}
	if httpInterceptor != nil {
		if response := httpInterceptor("GET", url, nil, h); response != nil {
			return *response, nil
		}
	}
	return parseResponse(c.client.R().SetHeaders(h).Get(url))
}

func (c DefaultHttpClient) PostForm(url string, formData FormData, headers *Headers) (Response, error) {
	h := Headers{}
	if headers != nil {
		h = *headers
	}
	if httpInterceptor != nil {
		if response := httpInterceptor("POST", url, formData, h); response != nil {
			return *response, nil
		}
	}
	return parseResponse(c.client.R().
		SetHeaders(h).
		SetFormData(formData).
		Post(url))

}

func (c DefaultHttpClient) Post(url string, body interface{}, headers *Headers) (Response, error) {
	h := Headers{}
	if headers != nil {
		h = *headers
	}
	if httpInterceptor != nil {
		if response := httpInterceptor("POST", url, body, h); response != nil {
			return *response, nil
		}
	}
	return parseResponse(c.client.R().
		SetHeaders(h).
		SetBody(body).
		Post(url))
}

func (c DefaultHttpClient) Delete(url string, body interface{}, headers *Headers) (Response, error) {
	h := Headers{}
	if headers != nil {
		h = *headers
	}
	if httpInterceptor != nil {
		if response := httpInterceptor("DELETE", url, body, h); response != nil {
			return *response, nil
		}
	}
	return parseResponse(c.client.R().
		SetHeaders(h).
		SetBody(body).
		Post(url))
}

func parseResponse(resp *resty.Response, err error) (Response, error) {

	if err != nil {
		return Response{}, err
	}

	if resp.IsError() {
		err = errors.Errorf("%s", resp.Body())
	}

	return Response{
		Status:  resp.StatusCode(),
		Body:    resp.Body(),
		IsError: resp.IsError(),
		Err: err,
	}, nil
}
