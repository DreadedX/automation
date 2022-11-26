package automation

import (
	"automation/device"
	"automation/home"
	"automation/integration/zigbee"
	"fmt"
	"log"

	paho "github.com/eclipse/paho.mqtt.golang"
)

func mixerAutomation(client paho.Client, prefix string, home *home.Home) {
	on(client, fmt.Sprintf("%s/living/remote"), func(message zigbee.RemoteState) {
		mixer, err := device.GetDevice[device.OnOff](&home.Devices, "living_room/mixer")
		if err != nil {
			log.Println(err)
			return
		}
		speakers, err := device.GetDevice[device.OnOff](&home.Devices, "living_room/speakers")
		if err != nil {
			log.Println(err)
			return
		}

		if message.Action == zigbee.ACTION_ON {
			if mixer.GetOnOff() {
				mixer.SetOnOff(false)
				speakers.SetOnOff(false)
			} else {
				mixer.SetOnOff(true)
			}
		} else if message.Action == zigbee.ACTION_BRIGHTNESS_UP {
			if speakers.GetOnOff() {
				speakers.SetOnOff(false)
			} else {
				speakers.SetOnOff(true)
				mixer.SetOnOff(true)
			}
		}
	})
}

