package main

import (
	"automation/automation"
	"automation/config"
	"automation/home"
	"automation/integration/hue"
	"automation/integration/kasa"
	"automation/integration/ntfy"
	"automation/integration/wol"
	"automation/integration/zigbee"
	"automation/presence"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	paho "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Get()

	// Setup all the connections to other services
	opts := paho.NewClientOptions().AddBroker(fmt.Sprintf("%s:%d", cfg.MQTT.Host, cfg.MQTT.Port))
	opts.SetClientID(cfg.MQTT.ClientID)
	opts.SetUsername(cfg.MQTT.Username)
	opts.SetPassword(cfg.MQTT.Password)
	opts.SetOrderMatters(false)

	client := paho.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	defer client.Disconnect(250)
	notify := ntfy.New(cfg.Ntfy.Topic)
	hue := hue.New(cfg.Hue.IP, cfg.Hue.Token)

	// Devices that we control and expose to google home
	home := home.New(cfg.Google.Username, cfg.Google.Credentials, client)

	// Setup presence system
	p := presence.New(client, hue, notify, home)
	defer p.Delete(client)

	r := mux.NewRouter()
	r.HandleFunc("/assistant", home.Service.FullfillmentHandler)

	// Register computers
	for name, info := range cfg.Computer {
		home.AddDevice(wol.NewComputer(info.MACAddress, name, info.Url))
	}

	// Register all kasa devies
	for name, ip := range cfg.Kasa.Outlets {
		home.AddDevice(kasa.NewOutlet(name, ip))
	}

	// Setup handler that automatically registers and updates all zigbee devices
	zigbee.DevicesHandler(client, home)
	automation.RegisterAutomations(client, hue, notify, home)

	addr := ":8090"
	srv := http.Server{
		Addr:    addr,
		Handler: r,
	}

	log.Printf("Starting server on %s (PID: %d)\n", addr, os.Getpid())
	srv.ListenAndServe()
}
