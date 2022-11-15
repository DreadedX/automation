package ntfy

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

type ntfy struct {
	topic string
}

func (ntfy *ntfy) Presence(home bool) {
	// @TODO Maybe add list the devices that are home currently?
	var description string
	var actions string
	if home {
		description = "Home"
		actions = "broadcast, Set as away, extras.cmd=presence, extras.state=0, clear=true"
	} else {
		description = "Away"
		actions = "broadcast, Set as home, extras.cmd=presence, extras.state=1, clear=true"
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://ntfy.sh/%s", ntfy.topic), strings.NewReader(description))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Title", "Presence")
	req.Header.Set("Tags", "house")
	req.Header.Set("Actions", actions)
	req.Header.Set("Priority", "1")

	http.DefaultClient.Do(req)
}

func Connect() ntfy {
	topic, _ := os.LookupEnv("NTFY_TOPIC")
	ntfy := ntfy{topic}

	// @TODO Make sure the topic is valid?

	return ntfy
}
