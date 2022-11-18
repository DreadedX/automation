package main

import (
	"automation/device"
	"automation/integration/hue"
	"automation/integration/kasa"
	"automation/integration/mqtt"
	"automation/integration/ntfy"
	"automation/presence"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

type config struct {
	Hue struct {
		Token string `yaml:"token" envconfig:"HUE_TOKEN"`
		IP    string `yaml:"ip" envconfig:"HUE_IP"`
	} `yaml:"hue"`

	NTFY struct {
		topic string `yaml:"topic" envconfig:"NTFY_TOPIC"`
	} `yaml:"ntfy"`

	MQTT struct {
		Host     string `yaml:"host" envconfig:"MQTT_HOST"`
		Port     int    `yaml:"port" envconfig:"MQTT_PORT"`
		Username string `yaml:"username" envconfig:"MQTT_USERNAME"`
		Password string `yaml:"password" envconfig:"MQTT_PASSWORD"`
		ClientID string `yaml:"client_id" envconfig:"MQTT_CLIENT_ID"`
	} `yaml:"mqtt"`

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

	// Setup all the connections to other services
	client := mqtt.New(config.MQTT.Host, config.MQTT.Port, config.MQTT.ClientID, config.MQTT.Username, config.MQTT.Password)
	defer client.Disconnect(250)
	notify := ntfy.New(config.NTFY.topic)
	hue := hue.New(config.Hue.IP, config.Hue.Token)

	// Setup presence system
	p := presence.New(client, hue, notify)
	defer p.Delete()

	// Register all kasa devies
	for name, ip := range config.Kasa.Outlets {
		devices[name] = kasa.New(ip)
	}

	// Devices that we control and expose to google home
	provider := device.NewProvider(config.Google, client)

	r := mux.NewRouter()
	r.HandleFunc("/assistant", provider.Service.FullfillmentHandler)

	for name, info := range config.Computer {
		provider.AddDevice(device.NewComputer(info.MACAddress, name, info.Room, info.Url))
	}

	// Presence

	addr := ":8090"
	srv := http.Server{
		Addr:    addr,
		Handler: r,
	}

	log.Printf("Starting server on %s (PID: %d)\n", addr, os.Getpid())
	srv.ListenAndServe()
}
