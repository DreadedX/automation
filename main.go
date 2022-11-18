package main

import (
	"automation/device"
	"automation/integration/hue"
	"automation/integration/kasa"
	"automation/integration/mqtt"
	"automation/integration/ntfy"
	"automation/presence"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type config struct {
	Hue  hue.Config  `yaml:"hue"`
	NTFY ntfy.Config `yaml:"ntfy"`
	MQTT mqtt.Config `yaml:"mqtt"`

	Kasa struct {
		Outlets map[string]string `yaml:"outlets"`
	} `yaml:"kasa"`

	Computer map[string]struct {
		MACAddress string `yaml:"mac"`
		Room       string `yaml:"room"`
		Url        string `yaml:"url"`
	} `yaml:"computers"`

	Google device.Config `yaml:"google"`
}

func GetConfig() config {
	// First load the config from the yaml file
	f, err := os.Open("config.yml")
	if err != nil {
		log.Fatalln("Failed to open config file", err)
	}
	defer f.Close()

	var cfg config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Fatalln("Failed to parse config file", err)
	}

	// Then load values from environment
	// This can be used to either override the config or pass in secrets
	err = envconfig.Process("", &cfg)
	if err != nil {
		log.Fatalln("Failed to parse environmet config", err)
	}

	return cfg
}

var devices map[string]interface{}

// // @TODO Implement this for the other devices as well
// func GetDeviceKasa(name string) (*kasa.Kasa, error) {
// 	deviceGeneric, ok := devices[name]
// 	if !ok {
// 		return nil, fmt.Errorf("Device does not exist")
// 	}

// 	device, ok := deviceGeneric.(kasa.Kasa)
// 	if !ok {
// 		return nil, fmt.Errorf("Device is not a Kasa device")
// 	}

// 	return &device, nil
// }

// func SetupBindings(m *mqtt.MQTT) {
// 	m.AddHandler("zigbee2mqtt/living_room/audio_remote", func(_ paho.Client, msg paho.Message) {
// 		mixer, err := GetDeviceKasa("living_room/mixer")
// 		if err != nil {
// 			log.Println(err)
// 			return
// 		}
// 		speakers, err := GetDeviceKasa("living_room/speakers")
// 		if err != nil {
// 			log.Println(err)
// 			return
// 		}

// 		var message struct {
// 			Action string `json:"action"`
// 		}
// 		err = json.Unmarshal(msg.Payload(), &message)
// 		if err != nil {
// 			log.Println(err)
// 			return
// 		}

// 		if message.Action == "on" {
// 			if mixer.GetState() {
// 				mixer.SetState(false)
// 				speakers.SetState(false)
// 			} else {
// 				mixer.SetState(true)
// 			}
// 		} else if message.Action == "brightness_move_up" {
// 			if speakers.GetState() {
// 				speakers.SetState(false)
// 			} else {
// 				speakers.SetState(true)
// 				mixer.SetState(true)
// 			}
// 		}
// 	})
// }

func main() {
	_ = godotenv.Load()

	config := GetConfig()

	devices = make(map[string]interface{})

	// MQTT
	m := mqtt.Connect(config.MQTT)
	defer m.Disconnect()

	// Hue
	h := hue.Connect(config.Hue)

	// Kasa
	for name, ip := range config.Kasa.Outlets {
		devices[name] = kasa.New(ip)
	}

	// ntfy.sh
	// n := ntfy.Connect(config.NTFY)

	// Devices that we control and expose to google home
	provider := device.NewProvider(config.Google, &m)

	r := mux.NewRouter()
	r.HandleFunc("/assistant", provider.Service.FullfillmentHandler)

	for name, info := range config.Computer {
		provider.AddDevice(device.NewComputer(info.MACAddress, name, info.Room, info.Url))
	}

	// Presence
	p := presence.New()
	m.AddHandler("automation/presence/+", p.PresenceHandler)

	m.AddHandler("automation/presence", func(client paho.Client, msg paho.Message) {
		if len(msg.Payload()) == 0 {
			// In this case we clear the persistent message
			return
		}
		var message presence.Message
		err := json.Unmarshal(msg.Payload(), &message)
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Printf("Presence: %t\n", message.State)
		// Notify users of presence update
		// n.Presence(present)

		// Set presence on the hue bridge
		h.SetFlag(41, message.State)

		if !message.State {
			// Turn off all the devices that we manage ourselves
			provider.TurnAllOff()

			// Turn off all devices
			// @TODO Maybe allow for exceptions, could be a list in the config that we check against?
			for _, device := range devices {
				switch d := device.(type) {
				case kasa.Kasa:
					d.SetState(false)

				}
			}

			// @TODO Turn off nest thermostat
		} else {
			// @TODO Turn on the nest thermostat again
		}
	})

	addr := ":8090"
	srv := http.Server{
		Addr:    addr,
		Handler: r,
	}

	log.Printf("Starting server on %s (PID: %d)\n", addr, os.Getpid())
	srv.ListenAndServe()
}
