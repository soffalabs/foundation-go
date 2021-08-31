package sentry

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/soffa-io/soffa-core-go/counters"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"time"
)

var sentryEnabled bool
var SentryExceptions = counters.NewCounter("x_app_sentry_exceptions", "Will track all exceptions", true)

func Init(dsn string, application string, version string) {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:         dsn,
		Environment: "prod",
		Release:     fmt.Sprintf("%s@%s", application, version),
		Debug:       false,
	})
	log.Default.FatalIf(err)
	defer sentry.Flush(2 * time.Second)
	sentryEnabled = true
	log.Default.Info("Sentry is enabled.")
}

func CaptureException(err error) {
	if err == nil {
		return
	}
	SentryExceptions.Inc()
	if sentryEnabled {
		sentry.CaptureException(err)
	}
}

func CaptureMessage(msg string, args ...interface{}) {
	if h.IsEmpty(msg) {
		return
	}
	if sentryEnabled {
		sentry.CaptureMessage(fmt.Sprintf(msg, args...))
	}
}
