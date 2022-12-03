package device

import (
	"fmt"
)

type Basic interface {
	GetID() InternalName
}

type OnOff interface {
	SetOnOff(state bool)
	GetOnOff() bool
}

type Activate interface {
	Activate(state bool)
}

func GetDevices[K any](devices *map[InternalName]Basic) map[InternalName]K {
	devs := make(map[InternalName]K)

	for name, device := range *devices {
		if dev, ok := device.(K); ok {
			devs[name] = dev
		}
	}

	return devs
}

func GetDevice[K any](devices *map[InternalName]Basic, name InternalName) (K, error) {
	d, ok := (*devices)[name]
	if !ok {
		var noop K
		return noop, fmt.Errorf("Device '%s' does not exist", name)
	}

	dev, ok := d.(K)
	if !ok {
		var noop K
		return noop, fmt.Errorf("Device '%s' is not the expected type", name)
	}

	return dev, nil
}

