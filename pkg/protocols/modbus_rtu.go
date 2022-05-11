package protocols

import (
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos"
	"reflect"
)

// TODO: 新增协议定义

const ModbusRtuKey = "modbus-rtu"

type ModbusRtuProtocols struct {
	Address     string `json:"Address"`
	BaudRate    string `json:"BaudRate"`
	DataBits    string `json:"DataBits"`
	StopBits    string `json:"StopBits"`
	Parity      string `json:"Parity"`
	UnitID      string `json:"UnitID"`
	Timeout     string `json:"Timeout"`
	IdleTimeout string `json:"IdleTimeout"`
}

func (p ModbusRtuProtocols) Struct2MapProtocolProperties() map[string]dtos.ProtocolProperties {
	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)
	var data = make(dtos.ProtocolProperties)
	for i := 0; i < t.NumField(); i++ {
		data[t.Field(i).Name] = v.Field(i).String()
	}
	return map[string]dtos.ProtocolProperties{
		ModbusRtuKey: data,
	}
}

// FormatParity 转换校验方式
// N（None [没有]） ------0
// O（Odd [单、奇、奇怪]）------1
// E (Even 偶、双、平均) -----2
// M（Mark 标记、符合）------3
// S（Space 空间、空地）------4
// Parity: N - None, O - Odd, E - Even
func (p ModbusRtuProtocols) FormatParity() string {
	switch p.Parity {
	case "1":
		return "O"
	case "2":
		return "E"
	case "3":
		return "M"
	case "4":
		return "S"
	default:
		return "N"
	}
}
