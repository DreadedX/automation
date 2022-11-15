package hue

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/r3labs/sse/v2"
)

type Hue struct {
	ip    string
	login string
	Events chan *sse.Event
}

func (hue *Hue) SetFlag(id int, value bool) {
	url := fmt.Sprintf("http://%s/api/%s/sensors/%d/state", hue.ip, hue.login, id)

	var data []byte
	if value {
		data = []byte(`{ "flag": true }`)
	} else {
		data = []byte(`{ "flag": false }`)
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}

	_, err = client.Do(req)
	if err != nil {
		panic(err)
	}
}

func Connect() Hue {
	login, _ := os.LookupEnv("HUE_BRIDGE")
	ip, _ := os.LookupEnv("HUE_IP")

	hue := Hue{ip: ip, login: login, Events: make(chan *sse.Event)}

	// Subscribe to eventstream
	client := sse.NewClient(fmt.Sprintf("https://%s/eventstream/clip/v2", hue.ip))
	client.Headers["hue-application-key"] = hue.login
	client.Connection.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	err := client.SubscribeChanRaw(hue.Events)
	if err != nil {
		panic(err)
	}

	return hue
}
