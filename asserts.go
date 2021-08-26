package sf

import (
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
)

func AssertNotEmpty(value string, message string) {
	if h.IsStrEmpty(value) {
		log.Fatal(message)
	}
}
