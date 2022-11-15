package google

import (
	"encoding/json"
)

type Intent string

const (
	IntentSync    Intent = "action.devices.SYNC"
	IntentQuery          = "action.devices.QUERY"
	IntentExecute        = "action.devices.EXECUTE"
)

type DeviceHandle struct {
	ID string `json:"id"`

	CustomData map[string]interface{} `json:"customData,omitempty"`
}

type queryPayload struct {
	Devices []DeviceHandle `json:"devices"`
}

type Command struct {
	Devices   []DeviceHandle `json:"devices"`
	Execution []Execution      `json:"execution"`
}

type executePayload struct {
	Commands []Command `json:"commands"`
}

type fullfilmentInput struct {
	Intent Intent

	Query   *queryPayload
	Execute *executePayload
}

type FullfillmentRequest struct {
	RequestID string             `json:"requestId"`
	Inputs    []fullfilmentInput `json:"inputs"`
}

func (i *fullfilmentInput) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Intent  Intent          `json:"intent"`
		Payload json.RawMessage `json:"payload"`
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	i.Intent = tmp.Intent
	switch i.Intent {
	case IntentQuery:
		payload := &queryPayload{}
		err = json.Unmarshal(tmp.Payload, payload)
		if err != nil {
			return err
		}
		i.Query = payload

	case IntentExecute:
		payload := &executePayload{}
		err = json.Unmarshal(tmp.Payload, payload)
		if err != nil {
			return err
		}
		i.Execute = payload
	}

	return nil
}
