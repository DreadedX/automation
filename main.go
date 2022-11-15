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
)

func main() {
	_ = godotenv.Load()

	// MQTT
	m := mqtt.Connect()
	defer m.Disconnect()

	// Hue
	h := hue.Connect()

	// Kasa
	mixer := kasa.New("10.0.0.49")
	speakers := kasa.New("10.0.0.182")

	// ntfy.sh
	n := ntfy.Connect()

	// Presence
	p := presence.New()
	m.AddHandler("automation/presence/+", p.PresenceHandler)

	// Smart home
	provider := device.NewProvider(&m)

	provider.AddDevice(device.NewComputer("30:9c:23:60:9c:13", "Zeus", "Living Room"))

	r := mux.NewRouter()
	r.HandleFunc("/assistant", provider.Service.FullfillmentHandler)

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
					mixer.SetState(false)
					speakers.SetState(false)

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
