package di
/*
import (
	"github.com/soffa-io/soffa-core-go/log"
	"go.uber.org/dig"
	"sync"
)
var (
	mu sync.Mutex
)
type Container struct {
	di *dig.Container
}

func New() *Container {
	return &Container{di: dig.New()}
}

func (c *Container) Inject(function interface{})  {
	log.Instance.FatalIf(c.di.Invoke(function))
}

func (c *Container) Provide(constructor interface{}) {
	mu.Lock()
	err := c.di.Provide(constructor)
	mu.Unlock()
	log.Instance.FatalIf(err)
}
*/