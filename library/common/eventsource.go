package common

import (
	"fmt"
	"time"

	"github.com/antage/eventsource"
)

func ExampleEventSource() {
	es := eventsource.New(nil, nil)
	es.ConsumersCount()
	defer es.Close()

	go func() {
		for {
			es.SendEventMessage(fmt.Sprintf("now time is %v", time.Now().String()), "", "")
			time.Sleep(time.Second * 2)
		}
	}()
}
