package hue

import (
	"time"
)

type EventType string

const (
	Update EventType = "update"
)

type DeviceType string

const (
	Light        DeviceType = "light"
	GroupedLight            = "grouped_light"
	Button                  = "button"
)

type LastEvent string

const (
	InitialPress LastEvent = "initial_press"
	ShortPress             = "short_press"
)

type device struct {
	ID    string `json:"id"`
	IDv1  string `json:"id_v1"`
	Owner struct {
		Rid   string `json:"rid"`
		Rtype string `json:"rtype"`
	} `json:"owner"`
	Type DeviceType `json:"type"`

	On *struct {
		On bool `json:"on"`
	} `json:"on"`

	Dimming *struct {
		Brightness float32 `json:"brightness"`
	} `json:"dimming"`

	ColorTemperature *struct {
		Mirek      int  `json:"mirek"`
		MirekValid bool `json:"mirek_valid"`
	} `json:"color_temperature"`

	Button *struct {
		LastEvent LastEvent `json:"last_event"`
	}
}

type Event struct {
	CreationTime time.Time `json:"creationtime"`
	Data         []device  `json:"data"`
	ID           string    `json:"id"`
	Type         EventType `json:"type"`
}
