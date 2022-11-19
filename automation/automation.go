package automation

import (
	"automation/home"
	"automation/integration/hue"
	"automation/integration/ntfy"

	paho "github.com/eclipse/paho.mqtt.golang"
)

func RegisterAutomations(client paho.Client, hue *hue.Hue, notify *ntfy.Notify, home *home.Home) {
	presenceAutomation(client, hue, notify, home)
	mixerAutomation(client, home)
}
