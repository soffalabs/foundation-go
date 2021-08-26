package sf

import (
	"github.com/soffa-io/soffa-core-go/commons"
	"github.com/soffa-io/soffa-core-go/log"
)

func AssertNotEmpty(value string, message string) {
	if commons.IsStrEmpty(value) {
		log.Fatal(message)
	}
}
