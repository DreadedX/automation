package main

import (
	"automation/hue"
	"automation/mqtt"
	"automation/ntfy"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func SendCmd(cmd []byte) {
}

func main() {
	_ = godotenv.Load()

	// Signals
	halt := make(chan os.Signal, 1)
	signal.Notify(halt, os.Interrupt, syscall.SIGTERM)

	// MQTT
	m := mqtt.Connect()
	defer m.Disconnect()

	// Hue
	h := hue.Connect()

	// Kasa
	// k := kasa.New("10.0.0.32")

	// ntfy.sh
	n := ntfy.Connect()

	// Event loop
	fmt.Println("Starting event loop")
events:
	for {
		select {
		case present := <-m.Presence:
			fmt.Printf("Presence: %t\n", present)
			h.SetFlag(41, present)
			n.Presence(present)

		case <-h.Events:
			break

		case <-halt:
			break events
		}
	}
}
