package sf

/*
import (
	"github.com/gin-gonic/gin"
)

var (
	httpClient = NewHttpClient(false)
)


func (inbound *Application) AddKongHealthcheck(url string) {
	inbound.AddToHealthcheck(ServiceCheck{
		Name: "Kong",
		Kind: "Url",
		Ping: func() error {
			resp, err := httpClient.Get(url, nil)
			if err != nil || resp.IsError {
				return AnyError(err, errors.Errorf("%s", resp.Body))
			}
			json := JsonValue{value: string(resp.Body)}
			if json.GetString("message", "") == "no Route matched with those values" {
				return nil
			}
			return errors.Errorf("expectation failed")
		},
	})
}

func (inbound *Application) AddStartupListener(callback func()) {
	if inbound.startupListeners == nil {
		inbound.startupListeners = []func(){}
	}
	inbound.startupListeners = append(inbound.startupListeners, callback)
}


func (inbound *Application) AddBrokerHealthcheck(name string, broker MessageBroker) {
	if inbound.messageBroker == nil {
		inbound.messageBroker = &broker
	}
	inbound.healthChecks = append(inbound.healthChecks, ServiceCheck{
		Name: name,
		Kind: "Broker",
		Ping: broker.Ping,
	})
}

func (inbound *Application) AddToHealthcheck(check ServiceCheck) {
	inbound.healthChecks = append(inbound.healthChecks, check)
}
*/
