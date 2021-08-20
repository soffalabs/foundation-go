package kong

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"github.com/soffa-io/soffa-core-go"
)

var client *resty.Client

type Client struct {
	BaseUrl string
}

type Consumer struct {
	Id string `json:"id"`
}

func init() {
	client = resty.New()
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
	resp, err := client.R().
		SetBody(sf.H{
			"username":  username,
			"custom_id": opts.CustomId,
			"tags":      opts.Tags,
		}).
		SetResult(consumer).
		Post(fmt.Sprintf("%s/consumers/", k.BaseUrl))

	if resp.StatusCode() == 200 || resp.StatusCode() == 201 {
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
	message := string(resp.Body())
	if err != nil {
		message = err.Error()
	}
	return nil, fmt.Errorf("kong.consumer.create.failed: %v", message)

}

func (k Client) GetConsumer(idOrName string) (*Consumer, error) {
	consumer := &Consumer{}
	resp, err := client.R().
		SetResult(consumer).
		Get(fmt.Sprintf("%s/consumers/%s", k.BaseUrl, idOrName))

	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == 200 {
		return consumer, nil
	}
	if resp.StatusCode() == 404 || resp.StatusCode() == 204 {
		return nil, nil
	}
	return nil, fmt.Errorf("[%d] %s", resp.StatusCode(), resp.Body())
}

func (k Client) SetConsumerKey(consumerId string, key string) error {
	url := fmt.Sprintf("%s/consumers/%s/key-auth", k.BaseUrl, consumerId)
	resp, err := client.R().
		SetBody(sf.H{"key": key}).
		Post(url)
	return parseResponse(resp, err)
}

func (k Client) SetConsumerBasicAuth(consumerId string, username string, password string) error {
	url := fmt.Sprintf("%s/consumers/%s/basic-auth", k.BaseUrl, consumerId)
	resp, err := client.R().
		SetBody(sf.H{"username": username, "password": password}).
		Post(url)
	return parseResponse(resp, err)
}

func parseResponse(resp *resty.Response, err error) error {
	if err != nil {
		return err
	}
	if resp.StatusCode() >= 400 {
		return fmt.Errorf("[%d] %s", resp.StatusCode(), resp.Body())
	}
	return nil
}

func (k Client) EnableGlobalKeyPlugin() error {
	return k.enableGlobalPlugin("key-auth", sf.H{
		"key_names": []string{"apikey"},
		"hide_credentials":true,
	})
}

func (k Client) EnableGlobalBasicAuth() error {
	return k.enableGlobalPlugin("basic-auth", sf.H{
		"hide_credentials":true,
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
	resp, err := client.R().
		SetBody(sf.H{
			"name":   name,
			"config": config,
		}).
		Post(fmt.Sprintf("%s/plugins/", k.BaseUrl))

	if err != nil {
		return err
	}
	if resp.StatusCode() == 409 {
		return nil
	}
	if resp.StatusCode() >= 300 {
		return fmt.Errorf("%d => %s", resp.StatusCode(), resp.Body())
	}
	log.Infof("kong %s plugin enabled.", name)
	return nil
}
