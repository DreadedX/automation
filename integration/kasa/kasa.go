package kasa

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
)

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

func decrypt(data []byte) string {
	var key byte = 171
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(len(data)))

	for _, c := range data {
		a := key ^ c
		key = c
		buf.WriteByte(a)
	}

	return string(buf.Bytes())
}


type Kasa struct {
	ip string
}

func New(ip string) Kasa {
	return Kasa{ip}
}

func (kasa *Kasa) sendCmd(cmd cmd) {
	con, err := net.Dial("tcp", fmt.Sprintf("%s:9999", kasa.ip))
	if err != nil {
		panic(err)
	}

	defer con.Close()

	b, err := json.Marshal(cmd)
	if err != nil {
		panic(err)
	}

	_, err = con.Write(encrypt(b))
	if err != nil {
		panic(err)
	}

	resp := make([]byte, 2048)
	_, err = con.Read(resp)
	if err != nil {
		panic(err)
	}

	var reply reply
	json.Unmarshal(resp, &reply)

	if reply.System.SetRelayState.ErrCode != 0 {
		fmt.Println(reply)
		fmt.Println(resp)
	}
}

func (kasa *Kasa) SetState(on bool) {
	var cmd cmd
	if on {
		cmd.System.SetRelayState.State = 1
	}

	kasa.sendCmd(cmd)
}
