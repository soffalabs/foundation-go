package sf

import (
	"crypto/tls"
	"github.com/go-resty/resty/v2"
	"time"
)

type HttpInterceptor = func(method string, url string, headers HttpHeaders, body interface{}) *HttpResponse
type HttpHeaders = map[string]string
type FormData = map[string]string
type HttpClient interface {
	PostForm(url string, headers HttpHeaders, formData FormData) (HttpResponse, error)
}
type HttpResponse struct {
	Status  int
	Body    []byte
	IsError bool
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

func NewHttpClient() HttpClient {
	client := resty.New()
	client.SetDebug(true)
	// client.SetTLSClientConfig(&tls.Config{ RootCAs: roots })
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: false})
	// Set client timeout as per your need
	client.SetTimeout(30 * time.Second)
	return DefaultHttpClient{
		client: client,
	}
}

func (c HttpResponse) Decode(dest interface{}) error {
	return FromJson(string(c.Body), dest)
}

func (c DefaultHttpClient) PostForm(url string, headers HttpHeaders, formData FormData) (HttpResponse, error) {

	if httpInterceptor != nil {
		if response := httpInterceptor("POST", url, headers, formData); response != nil {
			return *response, nil
		}
	}

	resp, err := c.client.R().
		SetHeaders(headers).
		SetFormData(formData).
		Post(url)

	if err != nil {
		return HttpResponse{}, err
	}

	return HttpResponse{
		Status:  resp.StatusCode(),
		Body:    resp.Body(),
		IsError: resp.IsError(),
	}, nil
}
