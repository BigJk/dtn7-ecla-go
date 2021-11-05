package ecla

import (
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type ECLA struct {
	moduleName string
	conn       *websocket.Conn
	done       chan struct{}

	fnOnBeacon      func(packet BeaconPacket)
	fnOnForwardData func(packet ForwardDataPacket)
}

func New(moduleName string) *ECLA {
	return &ECLA{
		moduleName: moduleName,
	}
}

func (ecla *ECLA) SetOnBeacon(fnOnBeacon func(packet BeaconPacket)) *ECLA {
	ecla.fnOnBeacon = fnOnBeacon
	return ecla
}

func (ecla *ECLA) SetOnForwardData(fnOnForwardData func(packet ForwardDataPacket)) *ECLA {
	ecla.fnOnForwardData = fnOnForwardData
	return ecla
}

func (ecla *ECLA) handler() {
	defer close(ecla.done)
	for {
		_, message, err := ecla.conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}

		var typePart TypePacket
		if err := json.Unmarshal(message, &typePart); err != nil {
			log.Println("packet type err:", err)
			return
		}

		switch typePart.Type {
		case BeaconPacketType:
			var beaconPacket BeaconPacket
			if err := json.Unmarshal(message, &beaconPacket); err != nil {
				log.Println("packet type err:", err)
				return
			}

			if ecla.fnOnBeacon != nil {
				ecla.fnOnBeacon(beaconPacket)
			}
		case ForwardDataPacketType:
			var forwardPacket ForwardDataPacket
			if err := json.Unmarshal(message, &forwardPacket); err != nil {
				log.Println("packet type err:", err)
				return
			}

			if ecla.fnOnForwardData != nil {
				ecla.fnOnForwardData(forwardPacket)
			}
		}
	}
}

func (ecla *ECLA) Dial(addr string) error {
	if ecla.done != nil {
		return errors.New("conn already opened once")
	}

	u := url.URL{Scheme: "ws", Host: addr, Path: ""}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	ecla.conn = c

	ecla.conn.SetPingHandler(nil)
	ecla.conn.SetPongHandler(nil)

	ecla.done = make(chan struct{})

	go ecla.handler()

	_ = c.WriteJSON(IdentPacket{
		Type: IdentPacketType,
		Name: ecla.moduleName,
	})

	return nil
}

func (ecla *ECLA) Close() error {
	err := ecla.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return err
	}

	select {
	case <-ecla.done:
	case <-time.After(time.Second):
		return errors.New("timeout")
	}
	return nil
}

func (ecla *ECLA) InsertBeaconPacket(packet BeaconPacket) {
	if ecla.conn == nil {
		return
	}

	_ = ecla.conn.WriteJSON(packet)
}

func (ecla *ECLA) InsertForwardDataPacket(packet ForwardDataPacket) {
	if ecla.conn == nil {
		return
	}

	_ = ecla.conn.WriteJSON(packet)
}
