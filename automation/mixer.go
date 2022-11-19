package automation

import (
	"automation/device"
	"automation/home"
	"encoding/json"
	"log"

	paho "github.com/eclipse/paho.mqtt.golang"
)

func mixerAutomation(client paho.Client, home *home.Home) {
	var handler paho.MessageHandler = func(client paho.Client, msg paho.Message) {
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

		var message struct {
			Action string `json:"action"`
		}
		err = json.Unmarshal(msg.Payload(), &message)
		if err != nil {
			log.Println(err)
			return
		}

		if message.Action == "on" {
			if mixer.GetOnOff() {
				mixer.SetOnOff(false)
				speakers.SetOnOff(false)
			} else {
				mixer.SetOnOff(true)
			}
		} else if message.Action == "brightness_move_up" {
			if speakers.GetOnOff() {
				speakers.SetOnOff(false)
			} else {
				speakers.SetOnOff(true)
				mixer.SetOnOff(true)
			}
		}
	}

	if token := client.Subscribe("test/remote", 1, handler); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}

