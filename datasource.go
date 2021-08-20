package sf

import (
	"fmt"
	"github.com/go-gormigrate/gormigrate/v2"
	log "github.com/sirupsen/logrus"
	"github.com/xo/dburl"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DbManager struct {
	primaryDatasource *DataSource
	datasourceFactory *DataSource
	datasources       []*DataSource
	datasourcesMap    map[string]*DataSource
	initialized       bool
}

type Repository interface {
	Create(model interface{}) error
	CreateSchema(name string) error
}

type DbLinkCallback = func(em DbLink) error

func (m *DbManager) GetPrimaryLink() DbLink {
	return m.primaryDatasource.dbLink
}

func (m *DbManager) GetLink(name string) DbLink {
	return m.datasourcesMap[name].dbLink
}

func (m *DbManager) withLink(tenantId string, cb DbLinkCallback) error {
	if ds, ok := m.datasourcesMap[tenantId]; ok {
		return cb(ds.dbLink)
	}
	if m.datasourceFactory != nil {
		db := m.datasourceFactory.dbLink
		return db.Transactional(func(tx DbLink) error {
			if err := tx.UseSchema(tenantId); err != nil {
				return err
			}
			return cb(tx)
		})
	}
	return fmt.Errorf("unable to find a database link")
}

func (m *DbManager) init() error {
	if m.initialized {
		return nil
	}
	m.datasourcesMap = map[string]*DataSource{}
	for index, ds := range m.datasources {
		em, err := CreateDbLink(ds)
		if err != nil {
			return err
		}
		m.datasources[index].dbLink = em
		m.datasourcesMap[ds.Name] = m.datasources[index]
	}
	m.initialized = true
	return nil
}

func (m *DbManager) migrate() error {
	if err := m.init(); err != nil {
		return err
	}
	if m.datasources == nil || len(m.datasources) == 0 {
		log.Info("no datasources defined.")
		return nil
	}
	log.Info("applying database migrations...")

	for _, ds := range m.datasources {
		if err := ds.applyMigrations(); err != nil {
			return fmt.Errorf("database migrations failed for [%s] with error %v", ds.Name, err.Error())
		}
	}
	log.Info("database migrations complete.")
	return nil
}

type DataSource struct {
	Name       string
	Url        string
	Primary    bool
	factory    bool
	schemas    []string
	dbLink     DbLink
	Migrations []*gormigrate.Migration
}

func (d DataSource) applyMigrations() error {
	if d.factory {
		log.Info("Factory datasource found, scanning all schemas")
		for _, schema := range d.schemas {
			if err := d.dbLink.ApplyMigrations(d.Migrations, &schema); err != nil {
				return err
			}
		}
		return nil
	} else {
		return d.dbLink.ApplyMigrations(d.Migrations, nil)
	}
}

type DatasourceLoader interface {
	LoadDatasources() ([]DataSource, error)
}

type FixedDatasourceLoader struct {
	DatasourceLoader
	Items []DataSource
}

type DatabaseTenantsLoader struct {
	DatasourceLoader
	Db    DbLink
	Query string
}

func (tl FixedDatasourceLoader) LoadDatasources() ([]DataSource, error) {
	return tl.Items, nil
}

func CreateDbLink(ds *DataSource) (DbLink, error) {

	if IsStrEmpty(ds.Url) {
		return nil, fmt.Errorf("invalid databaseUrl provided (empty)")
	}
	cnx, err := dburl.Parse(ds.Url)
	if err != nil {
		return nil, fmt.Errorf("error parsing databaseUrl: %v", err)
	}

	var dialect gorm.Dialector
	if cnx.Driver == "sqlite3" {
		dialect = sqlite.Open(cnx.DSN)
	} else if cnx.Driver == "postgres" {
		dialect = postgres.Open(cnx.DSN)
	} else {
		return nil, fmt.Errorf("Unsupported database dialect: %s", cnx.Driver)
	}
	link, err := gorm.Open(dialect, &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	return GormDbLink{Name: ds.Name, Connection: link}, nil
}

type DataSourceManagerBuilder struct {
	dm *DbManager
}

func NewDataSourceManagerBuilder() DataSourceManagerBuilder {
	return DataSourceManagerBuilder{
		dm: &DbManager{},
	}
}

func (b DataSourceManagerBuilder) SetPrimary(url string, migrations []*gormigrate.Migration) {
	ds := &DataSource{Name: "@", Url: url, Primary: true, Migrations: migrations}
	b.dm.primaryDatasource = ds
	b.dm.datasources = append(b.dm.datasources, ds)
}

func (b DataSourceManagerBuilder) Register(name string, url string, migrations []*gormigrate.Migration) {
	b.init()
	ds := &DataSource{
		Name:       name,
		Url:        url,
		Migrations: migrations,
	}
	b.dm.datasources = append(b.dm.datasources, ds)
}

func (b DataSourceManagerBuilder) init() {
	if b.dm.datasources == nil {
		b.dm.datasources = []*DataSource{}
	}
}

func (b DataSourceManagerBuilder) RegisterSchemaBaseTenants(url string, tenants []string, migrations []*gormigrate.Migration) {
	b.init()
	ds := &DataSource{
		Name:       "*",
		Url:        url,
		schemas:    tenants,
		factory:    true,
		Migrations: migrations,
	}
	b.dm.datasourceFactory = ds
	b.dm.datasources = append(b.dm.datasources, ds)
}

func (b DataSourceManagerBuilder) Get() *DbManager {
	return b.dm
}
