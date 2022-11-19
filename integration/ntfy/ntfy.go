package ntfy

import (
	"fmt"
	"net/http"
	"strings"
)

type Notify struct {
	topic string
}

func (n *Notify) Presence(home bool) {
	var description string
	var actions string
	if home {
		description = "Home"
		actions = "broadcast, Set as away, extras.cmd=presence, extras.state=0, clear=true"
	} else {
		description = "Away"
		actions = "broadcast, Set as home, extras.cmd=presence, extras.state=1, clear=true"
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://ntfy.sh/%s", n.topic), strings.NewReader(description))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Title", "Presence")
	req.Header.Set("Tags", "house")
	req.Header.Set("Actions", actions)
	req.Header.Set("Priority", "1")

	http.DefaultClient.Do(req)
}

func New(topic string) *Notify {
	ntfy := Notify{topic}

	// @TODO Make sure the topic is valid?

	return &ntfy
}
