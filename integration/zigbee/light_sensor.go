package zigbee

import (
	"automation/device"
	"encoding/json"
	"fmt"
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type lightSensor struct {
	info Info

	minValue int
	maxValue int
	timeout  time.Duration

	couldBeDark bool
	isDark      bool
	initialized bool
	timer       *time.Timer
}

type DarknessPayload struct {
	IsDark  bool  `json:"is_dark"`
	Updated int64 `json:"updated"`
}

func NewLightSensor(info Info, client paho.Client) *lightSensor {
	l := &lightSensor{info: info}

	// 1: 15 000 - 16 000 (Turns on to late)
	// 2: 22 000 - 30 000 (About 5-10 mins late)
	// 3: 23 000 - 30 000
	l.minValue = 23000
	l.maxValue = 25000

	l.timeout = 5 * time.Minute
	l.timer = time.NewTimer(l.timeout)
	l.timer.Stop()

	if token := client.Subscribe(l.info.MQTTAddress, 1, l.stateHandler); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}

	go func() {
		for {
			<-l.timer.C
			l.isDark = l.couldBeDark

			log.Println("Is dark:", l.isDark)

			payload, err := json.Marshal(DarknessPayload{
				IsDark:  l.isDark,
				Updated: time.Now().UnixMilli(),
			})
			if err != nil {
				log.Println(err)
			}

			if token := client.Publish(l.darknessTopic(), 1, true, payload); token.Wait() && token.Error() != nil {
				log.Println(token.Error())
			}
		}
	}()

	return l
}

func (l *lightSensor) darknessTopic() string {
	return fmt.Sprintf("automation/darkness/%s", l.info.FriendlyName.Room())
}

func (l *lightSensor) stateHandler(client paho.Client, msg paho.Message) {
	var message LightSensorState
	if err := json.Unmarshal(msg.Payload(), &message); err != nil {
		log.Println(err)
		return
	}

	fmt.Println(l.isDark, l.couldBeDark, message.Illuminance, l.maxValue, l.minValue)

	if !l.initialized {
		if message.Illuminance > l.maxValue {
			l.couldBeDark = false
		} else {
			l.couldBeDark = true
		}

		l.initialized = true
		l.timer.Reset(time.Millisecond)

		return
	}

	if message.Illuminance > l.maxValue {
		if l.isDark && l.couldBeDark {
			log.Println("Could be light, starting timer")
			l.couldBeDark = false
			l.timer.Reset(l.timeout)
		} else if !l.isDark {
			log.Println("Is not dark, canceling timer")
			l.couldBeDark = false
			l.timer.Stop()
		}
	} else if message.Illuminance < l.minValue {
		if !l.isDark && !l.couldBeDark {
			log.Println("Could be dark, starting timer")
			l.couldBeDark = true
			l.timer.Reset(l.timeout)
		} else if l.isDark {
			log.Println("Is dark, canceling timer")
			l.couldBeDark = true
			l.timer.Stop()
		}
	} else {
		// log.Println("In between the threshold, canceling timer for now keeping the current state")
		l.couldBeDark = l.isDark
		l.timer.Stop()
	}

}

// zigbee.Device
var _ Device = (*lightSensor)(nil)

func (l *lightSensor) IsZigbeeDevice() {}

func (l *lightSensor) Delete(client paho.Client) {
	if token := client.Unsubscribe(l.darknessTopic()); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}

// device.Base
var _ device.Basic = (*lightSensor)(nil)

func (l *lightSensor) GetID() device.InternalName {
	return l.info.FriendlyName
}
