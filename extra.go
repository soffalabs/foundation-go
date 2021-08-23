package soffa_core

/*
import (
	"github.com/gin-gonic/gin"
)

var (
	httpClient = NewHttpClient(false)
)


func (app *Application) AddKongHealthcheck(url string) {
	app.AddToHealthcheck(ServiceCheck{
		Name: "Kong",
		Kind: "Url",
		Ping: func() error {
			resp, err := httpClient.Get(url, nil)
			if err != nil || resp.IsError {
				return AnyError(err, fmt.Errorf("%s", resp.Body))
			}
			json := JsonValue{value: string(resp.Body)}
			if json.GetString("message", "") == "no Route matched with those values" {
				return nil
			}
			return fmt.Errorf("expectation failed")
		},
	})
}

func (app *Application) AddStartupListener(callback func()) {
	if app.startupListeners == nil {
		app.startupListeners = []func(){}
	}
	app.startupListeners = append(app.startupListeners, callback)
}


func (app *Application) AddBrokerHealthcheck(name string, broker MessageBroker) {
	if app.messageBroker == nil {
		app.messageBroker = &broker
	}
	app.healthChecks = append(app.healthChecks, ServiceCheck{
		Name: name,
		Kind: "Broker",
		Ping: broker.Ping,
	})
}

func (app *Application) AddToHealthcheck(check ServiceCheck) {
	app.healthChecks = append(app.healthChecks, check)
}
*/
