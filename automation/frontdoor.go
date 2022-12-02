package automation

import (
	"automation/presence"
	"encoding/json"
	"fmt"
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type Message struct {
	Contact bool `json:"contact"`
}

func frontdoorAutomation(client paho.Client, prefix string, p *presence.Presence) {
	const length = 15 * time.Minute

	timer := time.NewTimer(length)
	timer.Stop()

	on(client, fmt.Sprintf("%s/hallway/frontdoor", prefix), func(message Message) {
		// Always reset the timer if the door is opened
		if !message.Contact {
			timer.Reset(length)
		}

		// If the door opens an there is no one home
		if !message.Contact && !p.Current() {
			payload, err := json.Marshal(presence.Message{
				State:   true,
				Updated: time.Now().UnixMilli(),
			})
			if err != nil {
				log.Println(err)
			}

			token := client.Publish("automation/presence/frontdoor", 1, false, payload)
			if token.Wait() && token.Error() != nil {
				log.Println(token.Error())
			}
		}
	})

	go func() {
		for {
			<-timer.C
			// Clear out the value
			token := client.Publish("automation/presence/frontdoor", 1, false, "")
			if token.Wait() && token.Error() != nil {
				log.Println(token.Error())
			}
		}
	}()
}
