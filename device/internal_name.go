package device

import "strings"

type InternalName string

func (n InternalName) Room() string {
	s := strings.Split(string(n), "/")
	room := ""
	if len(s) > 1 {
		room = s[0]
	}
	room = strings.ReplaceAll(room, "_", " ")
	room = strings.Title(room)

	return room
}

func (n InternalName) Name() string {
	s := strings.Split(string(n), "/")
	name := s[0]
	if len(s) > 1 {
		name = s[1]
	}
	name = strings.Title(name)

	return name
}

func (n InternalName) String() string {
	return string(n)
}

