package soffa

import (
	"github.com/Masterminds/goutils"
	log "github.com/sirupsen/logrus"
	"os"
)

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
