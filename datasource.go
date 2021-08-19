package soffa

import (
	"fmt"
	"github.com/go-gormigrate/gormigrate/v2"
	log "github.com/sirupsen/logrus"
	"github.com/xo/dburl"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DatasourceManager struct {
	Multitenant bool
	Datasources []Datasource
	Migrations  []*gormigrate.Migration

	datasources map[string]Datasource
	initialized bool
}



func (m *DatasourceManager) Get() EntityManager {
	return m.Datasources[0].entityManager
}

func (m *DatasourceManager) GetTenant(tenantId string) *EntityManager {
	if ds, ok :=  m.datasources[tenantId]; ok {
		return &ds.entityManager
	}
	return nil
}

func (m *DatasourceManager) init() {
	if m.initialized {
		return
	}
	datasources := make(map[string]Datasource)
	for index, ds := range m.Datasources {
		em := CreateEntityManager(ds.Id, ds.Url, m.Migrations)
		m.Datasources[index].entityManager = em
		datasources[ds.Id] = m.Datasources[index]
	}
	m.datasources = datasources
	m.initialized = true
}

func (m *DatasourceManager) Migrate() error {
	m.init()
	if m.Migrations == nil || len(m.Migrations) == 0 {
		log.Info("no database migrations found.")
		return nil
	}
	if m.Datasources == nil || len(m.Datasources) == 0 {
		log.Info("no datasources defined.")
		return nil
	}
	log.Info("applying database migrations...")

	for _, ds := range m.Datasources {
		if err := ds.entityManager.ApplyMigrations(); err != nil {
			return fmt.Errorf("database migrations failed for [%s] with error %v", ds.Id, err.Error())
		}
	}
	log.Info("database migrations complete.")
	return nil
}

type Datasource struct {
	Id            string
	Url           string
	entityManager EntityManager
}

type TenantsLoader interface {
	LoadTenants() ([]Datasource, error)
}

type FixedTenantsLoader struct {
	TenantsLoader
	Tenants []string
}

type DatabaseTenantsLoader struct {
	TenantsLoader
	Db    EntityManager
	Query string
}

func (tl FixedTenantsLoader) LoadTenants() []string {
	return tl.Tenants
}

func (tl DatabaseTenantsLoader) LoadTenants() ([]string, error) {
	var tenants []string
	if err := tl.Db.Query(&tenants, tl.Query); err != nil {
		return nil, err
	}
	return tenants, nil
}

func CreateEntityManagers(sources []Datasource, migrations []*gormigrate.Migration) map[string]EntityManager {
	ds := make(map[string]EntityManager)
	for _, source := range sources {
		ds[source.Id] = CreateEntityManager(source.Id, source.Url, migrations)
	}
	return ds
}

func CreateEntityManager(name string, databaseUrl string, migrations []*gormigrate.Migration) EntityManager {

	if IsStrEmpty(databaseUrl) {
		log.Fatal("invalid databaseUrl provided (empty)")
		return nil
	}
	cnx, err := dburl.Parse(databaseUrl)
	if err != nil {
		log.Fatalf("Error parsing databaseUrl: %v", err)
		return nil
	}

	var dialect gorm.Dialector
	if cnx.Driver == "sqlite3" {
		dialect = sqlite.Open(cnx.DSN)
	} else if cnx.Driver == "postgres" {
		dialect = postgres.Open(cnx.DSN)
	} else {
		log.Fatalf("Unsupported database dialect: %s", cnx.Driver)
	}
	link, err := gorm.Open(dialect, &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	return GormEntityManager{Name: name, Link: link, migrations: migrations}
}
