package soffa

import (
	"github.com/go-gormigrate/gormigrate/v2"
	log "github.com/sirupsen/logrus"
	"github.com/xo/dburl"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)


type Migration struct {
	Author string
	Id string
	Changes []Change
}
//*gormigrate.Migration

type Change struct {}



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
