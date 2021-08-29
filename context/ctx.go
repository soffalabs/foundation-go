package context

import "context"

type Ctx struct {
	c context.Context
}

func (c *Ctx) Set(key string, value string) *Ctx {
	c.c = context.WithValue(c.c, key, value)
	return c
}

func (c *Ctx) Get(key string) interface{} {
	return c.c.Value(key)
}

func New() *Ctx {
	return &Ctx{c: context.Background()}
}

func (c *Ctx) Unwrap() context.Context {
	return c.c
}