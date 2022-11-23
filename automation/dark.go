package automation

import (
	"automation/integration/hue"
	"automation/integration/zigbee"

	paho "github.com/eclipse/paho.mqtt.golang"
)

func darknessAutomation(client paho.Client, hue *hue.Hue) {
	on(client, "automation/darkness/living", func(message zigbee.DarknessPayload) {
		hue.SetFlag(43, message.IsDark)
	})
}
