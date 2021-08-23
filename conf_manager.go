package soffa_core

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jeremywohl/flatten"
	"github.com/joho/godotenv"
	"github.com/soffa-io/soffa-core-go/log"
	"os"
	"strings"
)

type ConfManager struct {
	Env        string
	VaultUrl   string
	vaultData  map[string]interface{}
}

func newConfManager(env string) ConfManager {

	filenames := []string{fmt.Sprintf(".env.%s", strings.ToLower(env)), ".env"}
	for _, f := range filenames {
		if err := godotenv.Load(f); err == nil {
			log.Infof("%s file loaded", f)
		}
	}

	conf := ConfManager{
		Env:        strings.ToLower(env),
		VaultUrl:   os.Getenv("VAULT_URL"),
	}

	if !IsStrEmpty(conf.VaultUrl) {
		log.Infof("Loading config from vault: %s", conf.VaultUrl)
		data, err := ReadVaultSecret(conf.VaultUrl)
		if err != nil {
			log.Fatalf("Error starting service, failed to read secrets from vault.\n%v", err)
		} else {
			flat, err := flatten.Flatten(data, "", flatten.DotStyle)
			log.FatalErr(err)
			conf.vaultData = flat
		}
	}

	if conf.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}
	return conf
}

func (c ConfManager) IsProd() bool {
	return c.Env == "prod"
}

func (c ConfManager) IsTest() bool {
	return c.Env == "test"
}

func (c ConfManager) Get(paths ...string) string {
	for _, p := range paths {
		var value = ""
		if c.vaultData != nil {
			if res, ok := c.vaultData[p]; ok {
				value = fmt.Sprintf("%s", res)
			}
		}
		if IsStrEmpty(value) {
			value = os.Getenv(p)
		}
		if !IsStrEmpty(value) {
			return value
		}
	}
	return ""
}
