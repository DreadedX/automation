package google

type DeviceName struct {
	DefaultNames []string `json:"defaultNames,omitempty"`
	Name         string   `json:"name"`
	Nicknames    []string `json:"nicknames,omitempty"`
}

type DeviceInfo struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	Model        string `json:"model,omitempty"`
	HwVersion    string `json:"hwVersion,omitempty"`
	SwVersion    string `json:"swVersion,omitempty"`
}

type OtherDeviceID struct {
	AgentID  string `json:"agentId,omitempty"`
	DeviceID string `json:"deviceId,omitempty"`
}

type Device struct {
	ID   string `json:"id"`
	Type Type    `json:"type"`

	Traits []Trait `json:"traits"`

	Name DeviceName `json:"name"`

	WillReportState bool `json:"willReportState"`

	NotificationSupportedByAgent bool `json:"notificationSupportedByAgent,omitempty"`

	RoomHint string `json:"roomHint,omitempty"`

	DeviceInfo DeviceInfo `json:"deviceInfo,omitempty"`

	Attributes map[string]interface{} `json:"attributes,omitempty"`

	CustomData map[string]interface{} `json:"customDate,omitempty"`

	OtherDeviceIDs []OtherDeviceID `json:"otherDeviceIds,omitempty"`
}

func NewDevice(id string, typ Type) *Device {
	return &Device{
		ID:         id,
		Type:       typ,
		Attributes: make(map[string]interface{}),
		CustomData: make(map[string]interface{}),
	}
}
