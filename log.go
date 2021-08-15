package soffa

import (
	"github.com/Masterminds/goutils"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"os"
)

func InitApp(appName string) {
	if err := godotenv.Load(); err != nil {
		log.Warn(err)
	}
	InitLogging()

	env := os.Getenv("ENV")
	AppName = appName
	DevMode = env != "prod"
	if !DevMode {
		gin.SetMode(gin.ReleaseMode)
	}
	log.Infof("ENV = %s", env)
	log.Infof("DevMode = %v", DevMode)
}

func InitLogging() {
	log.SetOutput(os.Stdout)
	logLevel := os.Getenv("LOG_LEVEL")
	if goutils.IsEmpty(logLevel) {
		logLevel = "DEBUG"
	}
	if logLevel == "TRACE" {
		log.SetLevel(log.TraceLevel)
	} else if logLevel == "DEBUG" {
		log.SetLevel(log.DebugLevel)
	} else if logLevel == "WARN" {
		log.SetLevel(log.WarnLevel)
	} else if logLevel == "ERROR" {
		log.SetLevel(log.ErrorLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}
