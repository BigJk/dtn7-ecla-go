package ecla

const (
	RegisterPacketType    = "RegisterPacket"
	RegisteredPacketType  = "RegisteredPacket"
	ErrorPacketType       = "ErrorPacket"
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

type RegisteredPacket struct {
	Type   string      `json:"type"`
	EID    interface{} `json:"eid"`
	NodeID string      `json:"nodeid"`
}

type ErrorPacket struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
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
