package kasa

import (
	"automation/device"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
)

// This implementation is based on:
// https://www.softscheck.com/en/blog/tp-link-reverse-engineering/

type Device interface {
	device.Basic

	IsKasaDevice()
	GetIP() string
}

func encrypt(data []byte) []byte {
	var key byte = 171
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(len(data)))

	for _, c := range []byte(data) {
		a := key ^ c
		key = a
		buf.WriteByte(a)
	}

	return buf.Bytes()
}

func decrypt(data []byte) ([]byte, error) {
	var key byte = 171

	if len(data) < 4 {
		return nil, fmt.Errorf("Array has a minumun size of 4")
	}

	size := binary.BigEndian.Uint32(data[0:4])
	buf := make([]byte, size)

	for i := 0; i < int(size); i++ {
		a := key ^ data[i+4]
		key = data[i+4]
		buf[i] = a
	}

	return buf, nil
}

func sendCmd(kasa Device, cmd cmd) (reply, error) {
	con, err := net.Dial("tcp", fmt.Sprintf("%s:9999", kasa.GetIP()))
	if err != nil {
		return reply{}, err
	}

	defer con.Close()

	b, err := json.Marshal(cmd)
	if err != nil {
		return reply{}, err
	}

	_, err = con.Write(encrypt(b))
	if err != nil {
		return reply{}, err
	}

	resp := make([]byte, 2048)
	_, err = con.Read(resp)
	if err != nil {
		return reply{}, err
	}

	d, err := decrypt(resp)
	if err != nil {
		return reply{}, err
	}

	var reply reply
	err = json.Unmarshal(d, &reply)
	if err != nil {
		return reply, err
	}

	return reply, err
}

