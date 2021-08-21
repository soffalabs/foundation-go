package kong

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/soffa-io/soffa-core-go"
)

var httpClient sf.HttpClient

type Client struct {
	BaseUrl string
}

type Consumer struct {
	Id string `json:"id"`
}

func init() {
	httpClient = sf.NewHttpClient(false)
}

type ConsumerOptions struct {
	CustomId *string
	Tags     []string
	KeyAuth  string
	Login    string
	Password string
}

func (k Client) CreateConsumer(username string, opts *ConsumerOptions) (*Consumer, error) {
	consumer := &Consumer{}
	if opts == nil {
		opts = &ConsumerOptions{}
	}
	resp, err := httpClient.Post(fmt.Sprintf("%s/consumers/", k.BaseUrl), sf.H{
		"username":  username,
		"custom_id": opts.CustomId,
		"tags":      opts.Tags,
	}, nil)

	if err != nil || resp.IsError {
		return nil, sf.AnyError(err, resp.Err)
	}

	if err = resp.Decode(consumer); err != nil {
		return nil, err
	}
	if !sf.IsStrEmpty(opts.KeyAuth) {
		if err = k.EnableGlobalKeyPlugin(); err != nil {
			return nil, err
		}
		if err = k.SetConsumerKey(consumer.Id, opts.KeyAuth); err != nil {
			return nil, err
		}
	}
	if !sf.IsStrEmpty(opts.Login) && !sf.IsStrEmpty(opts.Password) {
		if err = k.EnableGlobalBasicAuth(); err != nil {
			return nil, err
		}
		if err = k.SetConsumerBasicAuth(consumer.Id, opts.Login, opts.Password); err != nil {
			return nil, err
		}
	}
	return consumer, nil
}

func (k Client) GetConsumer(idOrName string) (*Consumer, error) {
	consumer := &Consumer{}
	resp, err := httpClient.Get(fmt.Sprintf("%s/consumers/%s", k.BaseUrl, idOrName), nil)
	if err != nil || resp.IsError {
		return nil, sf.AnyError(err, resp.Err)
	}
	return consumer, nil
}

func (k Client) SetConsumerKey(consumerId string, key string) error {
	url := fmt.Sprintf("%s/consumers/%s/key-auth", k.BaseUrl, consumerId)
	resp, err := httpClient.Post(url, sf.H{"key": key}, nil)
	if err != nil || resp.IsError {
		return sf.AnyError(err, resp.Err)
	}
	return nil
}

func (k Client) SetConsumerBasicAuth(consumerId string, username string, password string) error {
	url := fmt.Sprintf("%s/consumers/%s/basic-auth", k.BaseUrl, consumerId)
	resp, err := httpClient.Post(url, sf.H{"username": username, "password": password}, nil)
	if err != nil || resp.IsError {
		return sf.AnyError(err, resp.Err)
	}
	return nil
}

func (k Client) EnableGlobalKeyPlugin() error {
	return k.enableGlobalPlugin("key-auth", sf.H{
		"key_names":        []string{"apikey"},
		"hide_credentials": true,
	})
}

func (k Client) EnableGlobalBasicAuth() error {
	return k.enableGlobalPlugin("basic-auth", sf.H{
		"hide_credentials": true,
	})
}

func (k Client) EnableGlobalCorsPlugin() error {
	return k.enableGlobalPlugin("cors", sf.H{
		"origins":            []string{"*"},
		"headers":            []string{"*"},
		"max_age":            3600,
		"preflight_continue": false,
		"credentials":        true,
		"methods":            []string{"GET", "POST"},
	})
}

func (k Client) enableGlobalPlugin(name string, config sf.H) error {
	resp, err := httpClient.Post(fmt.Sprintf("%s/plugins/", k.BaseUrl), sf.H{
		"name":   name,
		"config": config,
	}, nil)
	if err != nil || resp.IsError {
		if resp.Status == 409 {
			return nil
		}
		return sf.AnyError(err, resp.Err)
	}
	log.Infof("kong %s plugin enabled.", name)
	return nil
}
