package soffa

import (
	"github.com/go-gormigrate/gormigrate/v2"
	log "github.com/sirupsen/logrus"
	"github.com/xo/dburl"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func CreateEntityManager(databaseUrl string, migrations []*gormigrate.Migration) EntityManager {

	if IsStrEmpty(databaseUrl) {
		log.Fatal("invalid databaseUr provided")
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

	m := gormigrate.New(link, gormigrate.DefaultOptions, migrations)
	if err := m.Migrate(); err != nil{
		log.Fatalf("Could not migrate: %v", err)
	}else {
		log.Printf("Migration did run successfully")
	}

	return GormEntityManager{Link: link}
}
