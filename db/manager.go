package db

import (
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"regexp"
)

type Manager struct {
	ds          map[string]*DS
	migrated    bool
	serviceName string
}

func NewManager(serviceName string) *Manager {
	re := regexp.MustCompile("[^a-zA-Z0-9_]")
	serviceName = re.ReplaceAllString(serviceName, "_")
	return &Manager{
		ds:          map[string]*DS{},
		migrated:    false,
		serviceName: serviceName,
	}
}

func (m *Manager) Add(ds DS) *Link {
	if h.IsStrEmpty(ds.Id) {
		if !m.IsEmpty() {
			log.Default.Fatal("When adding multiple ds, an explicit Id is required")
		} else {
			ds.Id = "primary"
		}
	}
	if h.IsStrEmpty(ds.Url) {
		log.Default.Fatal("Database url cannot be empty")
	}
	ds.serviceName = m.serviceName
	ds.bootstrap()
	m.ds[ds.Id] = &ds
	return ds.link
}

func (m *Manager) Migrate() {
	if m.IsEmpty() || m.migrated {
		return
	}

	for _, el := range m.ds {
		log.Default.Infof("applying migrations on ds: %s", el.Id)
		el.migrate()
	}

	log.Default.Info("all migrationed were applied")
	m.migrated = true
}

func (m *Manager) Get(id string) *Link {
	return m.ds[id].link
}

func (m *Manager) GetLink() *Link {
	if len(m.ds) == 0 {
		errors.Raise(errors.New("No datasource configured"))
	}
	if len(m.ds) > 1 {
		errors.Raise(errors.New("More than 1 datasource configured, use GetLinkN() instead"))
	}
	for _, value := range m.ds {
		return value.link
	}
	return nil
}

func (m *Manager) GetLinkN(id string) *Link {
	ds, ok := m.ds[id]
	if ok {
		return ds.link
	}
	errors.Raise(errors.Errorf("invalid datasource id: %s", id))
	return nil
}

func (m *Manager) Ping() error {
	if m.ds == nil || len(m.ds) == 0 {
		return nil
	}
	for _, ds := range m.ds {
		if err := ds.ping(); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) IsEmpty() bool {
	return m.ds == nil || len(m.ds) == 0
}

func (m *Manager) Size() int {
	return len(m.ds)
}
