package main

import (
	"automation/integration/hue"
	"automation/integration/mqtt"
	"automation/integration/ntfy"
	"automation/device"
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
	// k := kasa.New("10.0.0.32")

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
				h.SetFlag(41, present)
				n.Presence(present)
				if !present {
					provider.TurnAllOff()
				} else {
					// In the future this is were we can do things like turning on the lights in the living room
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
