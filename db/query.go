package db

import "github.com/soffa-io/soffa-core-go/h"

type Query struct {
	subject  interface{}
	offset   int
	limit    int
	sort     interface{}
	args     []interface{}
	where    string
	whereMap *h.Map
}


type Result struct {
	Error        error
	Empty        bool
	RowsAffected int64
}

func Q() *Query {
	return &Query{offset: 0, limit: -1}
}

func (q *Query) Limit(value int) *Query {
	q.limit = value
	return q
}

func (q *Query) W(where h.Map) *Query {
	q.whereMap = &where
	return q
}

func (q *Query) Wheres(where string, args ...interface{}) *Query {
	q.where = where
	q.args = args
	return q
}

func (q *Query) Sort(field string) *Query {
	q.sort = field
	return q
}

