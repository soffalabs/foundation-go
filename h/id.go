package h

import (
	"fmt"
	"github.com/osamingo/indigo"
	"github.com/rs/xid"
	"github.com/soffa-io/soffa-core-go/errors"
	"time"
)

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
	t := time.Unix(1257894000, 0) // 2009-11-10 23:00:00 UTC
	g := indigo.New(nil, indigo.StartTime(t))
	value, err := g.NextID()
	errors.Raise(err)
	return value
}

