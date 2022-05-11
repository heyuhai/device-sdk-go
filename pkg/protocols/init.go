package protocols

// TODO: 新增

const (
	DeviceServiceNameByModbus = "device-modbus"
	VersionByModbus           = "v1.0.0"
)

const (
	ProtocolType645 = iota
	ProtocolTypeModbus
	ProtocolTypeMbPlcMail
	ProtocolTypeCharge645
	ProtocolTypeMidware645
)

// ProtocolType 协议类型
// PROTO_645，------0
// PROTO_MODBUS，------1
// PROTO_MB_PLCMAIL，--------2
// PROTO_CHARGE645，--------3
// PROTO_MIDWARE645 --------4
type ProtocolType int

func (p ProtocolType) String() string {
	switch p {
	case 1:
		return ModbusRtuKey
	default:
		return ""
	}
}

func (p ProtocolType) GetServiceName() string {
	switch p {
	case 1:
		return DeviceServiceNameByModbus
	default:
		return ""
	}
}

func (p ProtocolType) GetVersion() string {
	switch p {
	case 1:
		return VersionByModbus
	default:
		return ""
	}
}
