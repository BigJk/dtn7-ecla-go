package ecla

import (
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type State int

const (
	StateBeforeOpen   = State(0)
	StateConnected    = State(1)
	StateReconnecting = State(2)
)

type ECLA struct {
	sync.Mutex

	addr         string
	state        State
	moduleName   string
	enableBeacon bool
	conn         *websocket.Conn
	done         chan struct{}

	fnOnBeacon      func(packet BeaconPacket)
	fnOnForwardData func(packet ForwardDataPacket)
}

func New(moduleName string, enableBeacon bool) *ECLA {
	return &ECLA{
		state:        StateBeforeOpen,
		moduleName:   moduleName,
		enableBeacon: enableBeacon,
	}
}

func (ecla *ECLA) SetOnBeacon(fnOnBeacon func(packet BeaconPacket)) *ECLA {
	ecla.Lock()
	defer ecla.Unlock()

	ecla.fnOnBeacon = fnOnBeacon
	return ecla
}

func (ecla *ECLA) SetOnForwardData(fnOnForwardData func(packet ForwardDataPacket)) *ECLA {
	ecla.Lock()
	defer ecla.Unlock()

	ecla.fnOnForwardData = fnOnForwardData
	return ecla
}

func (ecla *ECLA) startReconnect() {
	ecla.state = StateReconnecting
}

func (ecla *ECLA) handler() {
	defer close(ecla.done)
	for {
		switch ecla.state {
		case StateReconnecting:
			log.Println("reconnecting...")

			if err := ecla.Dial(ecla.addr); err == nil {
				ecla.state = StateConnected
			} else {
				time.Sleep(time.Second)
			}
		case StateConnected:
			_, message, err := ecla.conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				ecla.startReconnect()
				continue
			}

			var typePart TypePacket
			if err := json.Unmarshal(message, &typePart); err != nil {
				log.Println("packet type err:", err)
				ecla.startReconnect()
				continue
			}

			switch typePart.Type {
			case BeaconPacketType:
				var beaconPacket BeaconPacket
				if err := json.Unmarshal(message, &beaconPacket); err != nil {
					log.Println("packet type err:", err)
					ecla.startReconnect()
					continue
				}

				if ecla.fnOnBeacon != nil {
					ecla.fnOnBeacon(beaconPacket)
				}
			case ForwardDataPacketType:
				var forwardPacket ForwardDataPacket
				if err := json.Unmarshal(message, &forwardPacket); err != nil {
					log.Println("packet type err:", err)
					ecla.startReconnect()
					continue
				}

				if ecla.fnOnForwardData != nil {
					ecla.fnOnForwardData(forwardPacket)
				}
			default:
				log.Println("unknown packet type:", typePart.Type)
			}
		}
	}
}

func (ecla *ECLA) Dial(addr string) error {
	ecla.Lock()
	defer ecla.Unlock()

	if ecla.state == StateConnected {
		return errors.New("conn already opened once")
	}

	ecla.addr = addr

	u := url.URL{Scheme: "ws", Host: addr, Path: ""}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	ecla.conn = c

	ecla.conn.SetPingHandler(nil)
	ecla.conn.SetPongHandler(nil)

	if ecla.state == StateBeforeOpen {
		ecla.done = make(chan struct{})
		ecla.state = StateConnected
		go ecla.handler()
	}

	_ = c.WriteJSON(RegisterPacket{
		Type:         RegisterPacketType,
		Name:         ecla.moduleName,
		EnableBeacon: ecla.enableBeacon,
	})

	return nil
}

func (ecla *ECLA) Close() error {
	ecla.Lock()
	defer ecla.Unlock()

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
	ecla.Lock()
	defer ecla.Unlock()

	if ecla.conn == nil {
		return
	}

	_ = ecla.conn.WriteJSON(packet)
}

func (ecla *ECLA) InsertForwardDataPacket(packet ForwardDataPacket) {
	ecla.Lock()
	defer ecla.Unlock()

	if ecla.conn == nil {
		return
	}

	_ = ecla.conn.WriteJSON(packet)
}
