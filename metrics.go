package sf

import "sync/atomic"

type Counter struct {
	Total    int32
	Success  int32
	Failures int32
}

func (a *Counter) Inc() {
	atomic.AddInt32(&a.Total, 1)
}

func (a *Counter) OK() {
	atomic.AddInt32(&a.Success, 1)
	atomic.AddInt32(&a.Total, 1)
}

func (a *Counter) KO() {
	atomic.AddInt32(&a.Failures, 1)
	atomic.AddInt32(&a.Total, 1)
}

func (a *Counter) CaptureFn(fn func() error) error {
	return a.Record(fn())
}

func (a *Counter) Record(err error) error {
	if err != nil {
		a.KO()
	} else {
		a.OK()
	}
	return err
}
