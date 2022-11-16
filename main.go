package main

import (
	"automation/device"
	"automation/integration/hue"
	"automation/integration/kasa"
	"automation/integration/mqtt"
	"automation/integration/ntfy"
	"automation/presence"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
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

func main() {
	_ = godotenv.Load()

	config := GetConfig()

	// MQTT
	m := mqtt.Connect(config.MQTT)
	defer m.Disconnect()

	// Hue
	h := hue.Connect(config.Hue)

	// Kasa
	kasaDevices := make(map[string]kasa.Kasa)
	for name, ip := range config.Kasa.Outlets {
		kasaDevices[name] = kasa.New(ip)
	}

	// ntfy.sh
	n := ntfy.Connect(config.NTFY)

	// Presence
	p := presence.New()
	m.AddHandler("automation/presence/+", p.PresenceHandler)

	// Devices that we control and expose to google home
	provider := device.NewProvider(config.Google, &m)

	r := mux.NewRouter()
	r.HandleFunc("/assistant", provider.Service.FullfillmentHandler)

	for name, info := range config.Computer {
		provider.AddDevice(device.NewComputer(info.MACAddress, name, info.Room, info.Url))
	}

	// Event loop
	go func() {
		fmt.Println("Starting event loop")
		for {
			select {
			case present := <-p.Presence:
				fmt.Printf("Presence: %t\n", present)
				// Notify users of presence update
				n.Presence(present)

				// Set presence on the hue bridge
				h.SetFlag(41, present)

				if !present {
					// Turn off all the devices that we manage ourselves
					provider.TurnAllOff()

					// Turn off kasa devices
					for _, device := range kasaDevices {
						device.SetState(false)
					}

					// @TODO Turn off nest thermostat
				} else {
					// @TODO Turn on the nest thermostat again
				}

			case <-h.Events:
				break
			}
		}
	}()

	addr := ":8090"
	srv := http.Server{
		Addr:    addr,
		Handler: r,
	}

	log.Printf("Starting server on %s (PID: %d)\n", addr, os.Getpid())
	srv.ListenAndServe()
}
