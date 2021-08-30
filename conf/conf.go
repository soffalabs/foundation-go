package conf

import (
	"fmt"
	"github.com/jeremywohl/flatten"
	"github.com/joho/godotenv"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"os"
	"strings"
)

type Manager struct {
	env        string
	vaultUrl   string
	vaultToken string
	vaultData  map[string]interface{}
	loaded     bool
}

func New(env string) *Manager {
	m := &Manager{env: strings.ToLower(env)}
	m.Load()
	return m
}

func UseDefault(env string) *Manager {
	m := &Manager{env: strings.ToLower(env)}
	m.vaultUrl = "auto"
	m.Load()
	return m
}

func (m *Manager) IsProdEnv() bool {
	return "prod" == m.env
}

func (m *Manager) IsTestEnv() bool {
	return "test" == m.env
}

func (m *Manager) UseVault(url string) {
	if h.IsEmpty(url) {
		log.Default.Fatal("Unable to locate vault url: env.VAULT_URL, env.VAULT_ADDR")
	}
	m.vaultUrl = url
	m.vaultToken = os.Getenv("VAULT_TOKEN")
}

func (m *Manager) Load() {

	if m.loaded {
		return
	}

	filenames := []string{fmt.Sprintf(".env.%s", strings.ToLower(m.env)), ".env"}
	for _, f := range filenames {
		if err := godotenv.Load(f); err == nil {
			log.Default.Infof("%s file loaded", f)
		}
	}

	if "auto" == m.vaultUrl {
		m.vaultUrl = os.Getenv("VAULT_URL")
		if h.IsEmpty(m.vaultUrl) {
			m.vaultUrl = os.Getenv("VAULT_ADDR")
		}
		m.vaultToken = os.Getenv("VAULT_TOKEN")
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}
	log.Default.SetLevel(logLevel)

	if h.IsStrEmpty(m.vaultUrl) {
		log.Default.Info("VaultUrl is empty skipping.")
	} else {
		log.Default.Infof("Loading config from vault: %s", m.vaultUrl)
		data, err := h.ReadVaultSecret(m.vaultUrl, m.vaultToken)
		if err != nil {
			log.Default.Fatalf("Error starting service, failed to read secrets from vault.\n%v", err)
		} else {
			flat, err := flatten.Flatten(data, "", flatten.DotStyle)
			log.Default.FatalIf(err)
			m.vaultData = flat
		}
	}

	m.loaded = true
	log.Default.Debug("Config manager loaded.")
}

func (m *Manager) Require(paths ...string) string {
	value := m.Get(paths...)
	if h.IsEmpty(value) {
		log.Default.Fatalf("[config] unable to locate one of: %s", strings.Join(paths, ","))
	}
	return value
}

func (m *Manager) Get(paths ...string) string {
	for _, p := range paths {
		var value = ""
		if m.vaultData != nil {
			if res, ok := m.vaultData[p]; ok {
				value = fmt.Sprintf("%s", res)
			}
		}
		if h.IsStrEmpty(value) {
			value = os.Getenv(p)
		}
		if !h.IsStrEmpty(value) {
			return value
		}
	}
	return ""
}
