package h

import (
	"github.com/soffa-io/soffa-core-go/log"
)

func AssertNotEmpty(value string, message string) {
	if IsStrEmpty(value) {
		log.Default.Fatal(message)
	}
}
