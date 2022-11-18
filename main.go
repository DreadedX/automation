package main

import (
	"automation/device"
	"automation/integration/hue"
	"automation/integration/kasa"
	"automation/integration/mqtt"
	"automation/integration/ntfy"
	"automation/presence"
	"encoding/json"
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

func SetupBindings(client paho.Client, p *device.Provider) {
	var handler paho.MessageHandler = func(client paho.Client, msg paho.Message) {
		mixer, err := p.Devices.GetKasaDevice("living_room/mixer")
		if err != nil {
			log.Println(err)
			return
		}
		speakers, err := p.Devices.GetKasaDevice("living_room/speakers")
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
			if mixer.GetState() {
				mixer.SetState(false)
				speakers.SetState(false)
			} else {
				mixer.SetState(true)
			}
		} else if message.Action == "brightness_move_up" {
			if speakers.GetState() {
				speakers.SetState(false)
			} else {
				speakers.SetState(true)
				mixer.SetState(true)
			}
		}
	}

	if token := client.Subscribe("test/remote", 1, handler); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}

func main() {
	_ = godotenv.Load()

	config := GetConfig()

	// Setup all the connections to other services
	client := mqtt.New(config.MQTT.Host, config.MQTT.Port, config.MQTT.ClientID, config.MQTT.Username, config.MQTT.Password)
	defer client.Disconnect(250)
	notify := ntfy.New(config.NTFY.topic)
	hue := hue.New(config.Hue.IP, config.Hue.Token)

	// Devices that we control and expose to google home
	provider := device.NewProvider(config.Google, client)

	// Setup presence system
	p := presence.New(client, hue, notify, provider)
	defer p.Delete()

	r := mux.NewRouter()
	r.HandleFunc("/assistant", provider.Service.FullfillmentHandler)

	// Register computers
	for name, info := range config.Computer {
		provider.AddDevice(device.NewComputer(info.MACAddress, name, info.Room, info.Url))
	}

	// Register all kasa devies
	for name, ip := range config.Kasa.Outlets {
		provider.AddDevice(kasa.New(name, ip))
	}

	SetupBindings(client, provider)

	// time.Sleep(time.Second)
	// pretty.Println(provider.Devices)
	// pretty.Println(provider.Devices.GetGoogleDevices())
	// pretty.Println(provider.Devices.GetKasaDevices())
	// pretty.Println(provider.Devices.GetZigbeeDevices())

	addr := ":8090"
	srv := http.Server{
		Addr:    addr,
		Handler: r,
	}

	log.Printf("Starting server on %s (PID: %d)\n", addr, os.Getpid())
	srv.ListenAndServe()
}
