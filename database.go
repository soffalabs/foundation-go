package soffa

import (
	"github.com/go-gormigrate/gormigrate/v2"
	log "github.com/sirupsen/logrus"
	"github.com/xo/dburl"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
)

func CreateEntityManager(migrations []*gormigrate.Migration) EntityManager {
	dbUrl := os.Getenv("DATABASE_URL")
	if len(dbUrl) == 0 {
		log.Fatal("missing DATABASE_URL")
	}
	cnx, err := dburl.Parse(dbUrl)
	if err != nil {
		log.Fatalf("error parinsg database url: %v", err)
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
