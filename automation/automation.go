package automation

import (
	"automation/home"
	"automation/integration/hue"
	"automation/integration/ntfy"
	"automation/presence"
	"encoding/json"
	"log"

	paho "github.com/eclipse/paho.mqtt.golang"
)

func on[M any](client paho.Client, topic string, onMessage func(message M)) {
	var handler paho.MessageHandler = func(c paho.Client, m paho.Message) {
		if len(m.Payload()) == 0 {
			// In this case we clear the persistent message
			// @TODO Maybe implement onClear as a callback? (Currently not needed)
			return
		}

		var message M;
		err := json.Unmarshal(m.Payload(), &message)
		if err != nil {
			log.Println(err)
			return
		}

		if onMessage != nil {
			onMessage(message)
		}
	}

	if token := client.Subscribe(topic, 1, handler); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}

func RegisterAutomations(client paho.Client, prefix string, hue *hue.Hue, notify *ntfy.Notify, home *home.Home, presence *presence.Presence) {
	presenceAutomation(client, hue, notify, home)
	mixerAutomation(client, prefix, home)
	kettleAutomation(client, prefix, home)
	darknessAutomation(client, hue)
	frontdoorAutomation(client, prefix, presence)
	zeusAutomation(client, home)
}
