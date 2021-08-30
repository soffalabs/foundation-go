package h

import (
	"fmt"
	"github.com/hashicorp/vault/api"
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/log"
	"net/url"
)

type VaultInterceptor = func() map[string]interface{}

var (
	vaultInterceptor *VaultInterceptor
)

func SetVaultInterceptor(fn VaultInterceptor) {
	vaultInterceptor = &fn
}

func ReadVaultSecret(uri string, token string) (map[string]interface{}, error) {

	var secret *api.Secret

	if vaultInterceptor != nil {
		data := (*vaultInterceptor)()
		secret = &api.Secret{Data: Map{"data": data}}
	} else {

		u, err := url.Parse(uri)
		if err != nil {
			log.Default.Errorf("url parsing failed: %s", uri)
			return nil, err
		}
		config := &api.Config{
			Address: fmt.Sprintf("%s://%s", u.Scheme, u.Host),
		}
		client, err := api.NewClient(config)
		if err != nil {
			return nil, err
		}
		if IsEmpty(token) {
			client.SetToken(u.User.Username())
		}else {
			client.SetToken(token)
		}
		secret, err = client.Logical().Read(fmt.Sprintf("secret/data/%s", u.Path))
		if err != nil {
			return nil, err
		}
		if secret == nil {
			return nil, errors.Errorf("unable to locate %s", u.Path)
		}
	}

	return secret.Data["data"].(map[string]interface{}), nil
}
