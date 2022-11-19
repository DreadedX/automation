package kasa

import (
	"automation/device"
	"log"
)

type Outlet struct {
	name device.InternalName
	ip string
}


func NewOutlet(name device.InternalName, ip string) *Outlet {
	return &Outlet{name, ip}
}

// kasa.Device
var _ Device = (*Outlet)(nil)
func (o *Outlet) GetIP() string {
	return o.ip
}

// device.Basic
var _ device.Basic = (*Outlet)(nil)
func (o *Outlet) GetID() device.InternalName {
	return o.name
}

// device.OnOff
var _ device.OnOff = (*Outlet)(nil)
func (o *Outlet) SetOnOff(on bool) {
	var cmd cmd
	cmd.System.SetRelayState = &SetRelayState{State: 0}
	if on {
		cmd.System.SetRelayState.State = 1
	}

	reply, err := sendCmd(o, cmd)
	if err != nil {
		log.Println(err)
		return
	}

	if reply.System.SetRelayState.ErrCode != 0 {
		log.Printf("Failed to set relay state, error: %d\n", reply.System.SetRelayState.ErrCode)
	}
}

func (o *Outlet) GetOnOff() bool {
	cmd := cmd{}

	cmd.System.GetSysinfo = &GetSysinfo{}
	
	reply, err := sendCmd(o, cmd)
	if err != nil {
		log.Println(err)
		return false
	}

	if reply.System.GetSysinfo.ErrCode != 0 {
		log.Printf("Failed to set relay state, error: %d\n", reply.System.GetSysinfo.ErrCode)

		return false
	}

	return reply.System.GetSysinfo.RelayState == 1
}
