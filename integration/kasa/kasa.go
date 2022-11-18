package kasa

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
)

// This implementation is based on:
// https://www.softscheck.com/en/blog/tp-link-reverse-engineering/

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


type Kasa struct {
	name string
	ip string
}

func New(name string, ip string) *Kasa {
	return &Kasa{name, ip}
}

func (kasa *Kasa) sendCmd(cmd cmd) (reply, error) {
	con, err := net.Dial("tcp", fmt.Sprintf("%s:9999", kasa.ip))
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

func (kasa *Kasa) SetState(on bool) {
	var cmd cmd
	cmd.System.SetRelayState = &SetRelayState{State: 0}
	if on {
		cmd.System.SetRelayState.State = 1
	}

	reply, err := kasa.sendCmd(cmd)
	if err != nil {
		log.Println(err)
		return
	}

	if reply.System.SetRelayState.ErrCode != 0 {
		log.Printf("Failed to set relay state, error: %d\n", reply.System.SetRelayState.ErrCode)
	}
}

func (kasa *Kasa) GetState() bool {
	cmd := cmd{}

	cmd.System.GetSysinfo = &GetSysinfo{}
	
	reply, err := kasa.sendCmd(cmd)
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

func (kasa *Kasa) GetName() string {
	return kasa.name
}

