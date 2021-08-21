package sf

import (
	"crypto/tls"
	"fmt"
	"github.com/go-resty/resty/v2"
	"time"
)

type HttpInterceptor = func(method string, url string, body interface{}, headers HttpHeaders) *HttpResponse
type HttpHeaders = map[string]string
type FormData = map[string]string

type HttpClient interface {
	Get(url string, headers *HttpHeaders) (HttpResponse, error)
	PostForm(url string, formData FormData, headers *HttpHeaders) (HttpResponse, error)
	Post(url string, payload interface{}, headers *HttpHeaders) (HttpResponse, error)
}

type HttpResponse struct {
	Status  int
	Body    []byte
	IsError bool
	Err     error
}
type DefaultHttpClient struct {
	client *resty.Client
}

var (
	httpInterceptor HttpInterceptor
)

func RegisterHttpInterceptor(interceptor HttpInterceptor) {
	httpInterceptor = interceptor
}

func NewHttpClient(debug bool) HttpClient {
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

func (c HttpResponse) Decode(dest interface{}) error {
	return FromJson(string(c.Body), dest)
}

func (c DefaultHttpClient) Get(url string, headers *HttpHeaders) (HttpResponse, error) {
	h := HttpHeaders{}
	if headers != nil {
		h = *headers
	}
	return parseResponse(c.client.R().SetHeaders(h).Get(url))
}

func (c DefaultHttpClient) PostForm(url string, formData FormData, headers *HttpHeaders) (HttpResponse, error) {
	h := HttpHeaders{}
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

func (c DefaultHttpClient) Post(url string, body interface{}, headers *HttpHeaders) (HttpResponse, error) {
	h := HttpHeaders{}
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

func parseResponse(resp *resty.Response, err error) (HttpResponse, error) {

	if err != nil {
		return HttpResponse{}, err
	}

	if resp.IsError() {
		err = fmt.Errorf("%s", resp.Body())
	}

	return HttpResponse{
		Status:  resp.StatusCode(),
		Body:    resp.Body(),
		IsError: resp.IsError(),
		Err: err,
	}, nil
}
