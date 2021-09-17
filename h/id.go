package h

import (
	"fmt"
	"github.com/osamingo/indigo"
	"github.com/rs/xid"
	"github.com/soffa-io/soffa-core-go/errors"
	"log"
	"time"
)

var g *indigo.Generator

func init() {
	t := time.Unix(1630698318, 0) // 2021-09-03 20:45:00 UTC
	g = indigo.New(nil, indigo.StartTime(t))
	_, err := g.NextID()
	if err != nil {
		log.Fatalln(err)
	}
}

// NewUniqueIdP Create a new UniqueId with a prefix
func NewUniqueIdP(prefix string) string {
	return fmt.Sprintf("%s%s", prefix, xid.New().String())
}

// NewUniqueId Create a new UniqueId
func NewUniqueId() string {
	return NewUniqueIdP("")
}

// NewShortId Create a new UniqueId
func NewShortId() string {
	value, err := g.NextID()
	errors.Raise(err)
	return value
}
