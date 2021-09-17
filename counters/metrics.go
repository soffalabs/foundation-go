package counters

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/soffa-io/soffa-core-go/conf"
	"sync/atomic"
)

type Counter struct {
	ok      int32
	err     int32
	code    string
	desc    string
	pmTotal prometheus.Counter
	pmErr   prometheus.Counter
	export  bool
}

var (
	__registry = map[string]*Counter{}
)

func NewCounter(code string, desc string, export bool) *Counter {
	if counter, ok := __registry[code]; ok {
		return counter
	}
	counter := &Counter{
		code:   code,
		desc:   desc,
		export: export,
	}
	__registry[code] = counter
	return counter
}

func (a *Counter) Record(err error) {
	if err == nil {
		a.Inc()
	} else {
		a.Err()
	}
}

func (a *Counter) Reset() {
	atomic.StoreInt32(&a.ok, 0)
	atomic.StoreInt32(&a.err, 0)
}

func (a *Counter) Watch(cb func() error) error {
	err := cb()
	a.Record(err)
	return err
}

func (a *Counter) Inc() {
	if a.export {
		if a.pmTotal == nil && conf.PrometheusEnabled {
			a.pmTotal = promauto.NewCounter(prometheus.CounterOpts{
				Name: a.code,
				Help: a.desc,
			})
		}
		if a.pmTotal != nil {
			a.pmTotal.Inc()
		}
	}
	atomic.AddInt32(&a.ok, 1)
}

func (a *Counter) Err() {
	if a.export {
		if a.pmErr == nil && conf.PrometheusEnabled {
			a.pmErr = promauto.NewCounter(prometheus.CounterOpts{
				Name: a.code + "_errors",
				Help: a.desc,
			})
		}
		if a.pmErr != nil {
			a.pmErr.Inc()
		}
	}
	atomic.AddInt32(&a.err, 1)
}

func (a *Counter) Success() int32 {
	return a.ok
}

func (a *Counter) Errors() int32 {
	return a.err
}

func (a *Counter) Total() int32 {
	return a.err + a.ok
}

func (a *Counter) Recover(rec interface{}, rethrow bool) {
	if rec == nil {
		a.Inc()
	} else {
		a.Err()
		if rethrow {
			panic(rec)
		}
	}
}
