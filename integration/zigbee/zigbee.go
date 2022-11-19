package zigbee

import "automation/device"

type Info struct {
	IEEEAdress      string       `json:"ieee_address"`
	FriendlyName    device.InternalName `json:"friendly_name"`
	Description     string       `json:"description"`
	Manufacturer    string       `json:"manufacturer"`
	ModelID         string       `json:"model_id"`
	SoftwareBuildID string       `json:"software_build_id"`
}

type Device interface {
	device.Basic

	IsZigbeeDevice()
	Delete()
}
