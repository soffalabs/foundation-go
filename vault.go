package sf

import (
	"fmt"
	"github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
	"net/url"
)

func ReadVaultSecret(uri string, dest interface{}) error {
	u, err := url.Parse(uri)
	if err != nil {
		log.Errorf("url parsing failed: %s", uri)
		return err
	}
	config := &api.Config{
		Address: fmt.Sprintf("%s://%s", u.Scheme, u.Host),
	}
	client, err := api.NewClient(config)
	if err != nil {
		return err
	}
	client.SetToken(u.User.Username())
	secret, err := client.Logical().Read(fmt.Sprintf("secret/data/%s", u.Path))
	if err != nil {
		return err
	}
	if secret == nil {
		return fmt.Errorf("unable to locate %s", u.Path)
	}
	return Convert(secret.Data["data"], dest)

}
