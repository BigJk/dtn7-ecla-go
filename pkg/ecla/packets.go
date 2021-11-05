package ecla

const (
	IdentPacketType       = "IdentPacket"
	BeaconPacketType      = "Beacon"
	ForwardDataPacketType = "ForwardDataPacket"
)

type TypePacket struct {
	Type string `json:"type"`
}

type IdentPacket struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type BeaconPacket struct {
	Type         string      `json:"type"`
	Addr         string      `json:"addr"`
	EID          interface{} `json:"eid"`
	ServiceBlock []int       `json:"service_block"`
}

type ForwardDataPacket struct {
	Type string `json:"type"`
	To   string `json:"to"`
	From string `json:"from"`
	Data []int  `json:"data"`
}

type ReceiveDataPacket struct {
	Type string `json:"type"`
	From string `json:"from"`
	Data []int  `json:"data"`
}
