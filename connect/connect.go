package connect

import (
	"automation/integration/hue"
	"automation/integration/ntfy"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type Connect struct {
	Client paho.Client
	Hue hue.Hue
	Notify ntfy.Notify
}
