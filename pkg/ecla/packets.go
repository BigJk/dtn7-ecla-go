package ecla

const (
	RegisterPacketType    = "RegisterPacket"
	BeaconPacketType      = "Beacon"
	ForwardDataPacketType = "ForwardDataPacket"
)

type TypePacket struct {
	Type string `json:"type"`
}

type RegisterPacket struct {
	Type         string `json:"type"`
	Name         string `json:"name"`
	EnableBeacon bool   `json:"enable_beacon"`
}

type BeaconPacket struct {
	Type         string      `json:"type"`
	Addr         string      `json:"addr"`
	EID          interface{} `json:"eid"`
	ServiceBlock string      `json:"service_block"`
}

type ForwardDataPacket struct {
	Type string `json:"type"`
	Dst  string `json:"dst"`
	Src  string `json:"src"`
	Data string `json:"data"`
}
