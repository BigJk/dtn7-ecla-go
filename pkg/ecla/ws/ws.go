package ws

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/BigJk/dtn7-ecla-go/pkg/ecla"
	"github.com/gorilla/websocket"
	"math/rand"
	"net/url"
	"sync"
	"time"
)

type State int

const (
	StateBeforeOpen         = State(0)
	StateBeforeRegistration = State(1)
	StateConnected          = State(2)
	StateReconnecting       = State(3)
	StateError              = State(4)
)

var stateNames = []string{"StateBeforeOpen", "StateBeforeRegistration", "StateConnected", "StateReconnecting", "StateError"}

type ECLA struct {
	sync.Mutex

	addr         string
	state        State
	moduleName   string
	enableBeacon bool
	conn         *websocket.Conn
	err          error
	done         chan struct{}
	wg           sync.WaitGroup

	allowReconnect        bool
	allowInitialReconnect bool

	fnOnBeacon      func(packet ecla.BeaconPacket)
	fnOnForwardData func(packet ecla.ForwardDataPacket)
	fnOnRegistered  func(packet ecla.RegisteredPacket)
	fnOnIdRequest   func() string

	logger ecla.Logger
}

func New(moduleName string, enableBeacon bool) *ECLA {
	randomId := fmt.Sprint(rand.Int63())

	return &ECLA{
		state:        StateBeforeOpen,
		moduleName:   moduleName,
		enableBeacon: enableBeacon,
		fnOnBeacon: func(ecla.BeaconPacket) {

		},
		fnOnForwardData: func(ecla.ForwardDataPacket) {

		},
		fnOnRegistered: func(ecla.RegisteredPacket) {

		},
		fnOnIdRequest: func() string {
			return randomId
		},
		logger: &ecla.EmptyLogger{},
	}
}

func (e *ECLA) WithLogger(logger ecla.Logger) *ECLA {
	e.Lock()
	defer e.Unlock()

	e.logger = logger
	return e
}

func (e *ECLA) WithOnBeacon(fnOnBeacon func(packet ecla.BeaconPacket)) *ECLA {
	e.Lock()
	defer e.Unlock()

	e.fnOnBeacon = fnOnBeacon
	return e
}

func (e *ECLA) WithOnForwardData(fnOnForwardData func(packet ecla.ForwardDataPacket)) *ECLA {
	e.Lock()
	defer e.Unlock()

	e.fnOnForwardData = fnOnForwardData
	return e
}

func (e *ECLA) WithOnRegistered(fnOnRegistered func(packet ecla.RegisteredPacket)) *ECLA {
	e.Lock()
	defer e.Unlock()

	e.fnOnRegistered = fnOnRegistered
	return e
}

func (e *ECLA) WithOnIdRequest(fnOnIdRequest func() string) *ECLA {
	e.Lock()
	defer e.Unlock()

	e.fnOnIdRequest = fnOnIdRequest
	return e
}

func (e *ECLA) WithReconnect(state bool) *ECLA {
	e.Lock()
	defer e.Unlock()

	e.allowReconnect = state
	return e
}

func (e *ECLA) WithInitialReconnect(state bool) *ECLA {
	e.Lock()
	defer e.Unlock()

	e.allowInitialReconnect = state
	return e
}

func (e *ECLA) readNextMessage() ([]byte, string, error) {
	e.Lock()
	defer e.Unlock()

	_, message, err := e.conn.ReadMessage()
	if err != nil {
		return nil, "", err
	}

	var typePart ecla.TypePacket
	if err := json.Unmarshal(message, &typePart); err != nil {
		return nil, "", err
	}

	return message, typePart.Type, nil
}

func (e *ECLA) openConn(addr string) (*websocket.Conn, error) {
	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws/ecla"}
	e.logger.Logf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	return c, err
}

func (e *ECLA) setConn(conn *websocket.Conn) {
	e.Lock()
	e.Unlock()

	e.conn = conn
	e.conn.SetPingHandler(nil)
	e.conn.SetPongHandler(nil)
}

func (e *ECLA) transition(newState State) {
	e.Lock()
	e.Unlock()

	old := e.state
	e.state = newState

	e.logger.Logf("Moving from '%s' to '%s'\n", stateNames[old], stateNames[newState])
}

func (e *ECLA) parseRegisteredPacket(data []byte) (ecla.RegisteredPacket, error) {
	var p ecla.RegisteredPacket
	if err := json.Unmarshal(data, &p); err != nil {
		e.logger.Log("error while unmarshalling packet (" + err.Error() + ")")
		return ecla.RegisteredPacket{}, err
	}
	return p, nil
}

func (e *ECLA) parseBeaconPacket(data []byte) (ecla.BeaconPacket, error) {
	var p ecla.BeaconPacket
	if err := json.Unmarshal(data, &p); err != nil {
		e.logger.Log("error while unmarshalling packet (" + err.Error() + ")")
		return ecla.BeaconPacket{}, err
	}
	return p, nil
}

func (e *ECLA) parseForwardDataPacket(data []byte) (ecla.ForwardDataPacket, error) {
	var p ecla.ForwardDataPacket
	if err := json.Unmarshal(data, &p); err != nil {
		e.logger.Log("error while unmarshalling packet (" + err.Error() + ")")
		return ecla.ForwardDataPacket{}, err
	}
	return p, nil
}

func (e *ECLA) parseErrorPacket(data []byte) (ecla.ErrorPacket, error) {
	var p ecla.ErrorPacket
	if err := json.Unmarshal(data, &p); err != nil {
		e.logger.Log("error while unmarshalling packet (" + err.Error() + ")")
		return ecla.ErrorPacket{}, err
	}
	return p, nil
}

func (e *ECLA) Dial(addr string) error {
	e.wg.Add(1)
	defer e.wg.Done()

	for {
		select {
		case <-e.done:
			return nil
		default:
			switch e.state {
			case StateBeforeOpen:
				c, err := e.openConn(addr)
				if err != nil {
					if e.allowInitialReconnect {
						e.logger.Log("connecting unsuccessful... Trying again in 1s")
						time.Sleep(time.Second)
						continue
					}
					return err
				}
				e.setConn(c)

				_ = c.WriteJSON(ecla.RegisterPacket{
					Type:         ecla.RegisterPacketType,
					Name:         e.moduleName,
					EnableBeacon: e.enableBeacon,
				})

				e.transition(StateBeforeRegistration)
			case StateBeforeRegistration:
				data, t, err := e.readNextMessage()
				if err != nil {
					e.state = StateReconnecting
					continue
				}

				if t == ecla.RegisteredPacketType { // RegisteredPacket Success
					reg, err := e.parseRegisteredPacket(data)
					if err != nil {
						continue
					}

					e.fnOnRegistered(reg)

					e.transition(StateConnected)
				} else if t == ecla.ErrorPacketType { // ErrorPacket
					errPack, err := e.parseErrorPacket(data)
					if err != nil {
						continue
					}

					e.err = errors.New(errPack.Reason)

					e.transition(StateError)
				} else { // Wrong Packet
					e.logger.Log("wrong packet received after registration (" + t + ")")
				}
			case StateConnected:
				data, t, err := e.readNextMessage()
				if err != nil {
					if e.allowReconnect {
						e.transition(StateReconnecting)
						continue
					}

					return err
				}

				switch t {
				case ecla.ErrorPacketType:
					errPack, err := e.parseErrorPacket(data)
					if err != nil {
						continue
					}

					e.err = errors.New(errPack.Reason)

					e.transition(StateError)
				case ecla.BeaconPacketType:
					b, err := e.parseBeaconPacket(data)
					if err != nil {
						continue
					}

					b.Addr = e.fnOnIdRequest()
					e.fnOnBeacon(b)
				case ecla.ForwardDataPacketType:
					fwd, err := e.parseForwardDataPacket(data)
					if err != nil {
						continue
					}

					fwd.Src = e.fnOnIdRequest()
					e.fnOnForwardData(fwd)
				}

			case StateReconnecting:
				c, err := e.openConn(addr)
				if err != nil {
					e.logger.Log("reconnecting unsuccessful... Trying again in 1s")
					time.Sleep(time.Second)
					continue
				}

				e.setConn(c)

				e.transition(StateBeforeRegistration)

				_ = c.WriteJSON(ecla.RegisterPacket{
					Type:         ecla.RegisterPacketType,
					Name:         e.moduleName,
					EnableBeacon: e.enableBeacon,
				})
			case StateError:
				return e.err
			}
		}
	}
}

func (e *ECLA) InsertBeaconPacket(packet ecla.BeaconPacket) {
	e.Lock()
	defer e.Unlock()

	if e.conn == nil {
		return
	}

	_ = e.conn.WriteJSON(packet)
}

func (e *ECLA) InsertForwardDataPacket(packet ecla.ForwardDataPacket) {
	e.Lock()
	defer e.Unlock()

	if e.conn == nil {
		return
	}

	_ = e.conn.WriteJSON(packet)
}

func (e *ECLA) Close() error {
	e.Lock()
	err := e.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return err
	}
	if e.conn != nil {
		return e.conn.Close()
	}
	e.Unlock()

	e.done <- struct{}{}
	e.wg.Wait()

	e.logger.Log("closed")

	return nil
}
