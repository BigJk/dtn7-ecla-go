package main

import (
	"encoding/json"
	"fmt"
	"github.com/BigJk/dtn7-ecla-go/pkg/ecla/ws"
	"log"
	"math/rand"
	"os"
	"os/signal"

	"github.com/BigJk/dtn7-ecla-go/pkg/ecla"
	"github.com/davecgh/go-spew/spew"
	"github.com/nats-io/nats.go"
)

func main() {
	// Generate some random id
	id := fmt.Sprintf("%d", rand.Int63())

	// Open NATS Connection
	nc, err := nats.Connect(os.Getenv("NATS_URL"), nats.Token(os.Getenv("NATS_TOKEN")))
	if err != nil {
		log.Fatal(err)
	}

	// Create ECLA
	ec := ws.New("NATS", true).WithOnBeacon(func(packet ecla.BeaconPacket) {
		packet.Addr = id

		fmt.Println("== [ECLA] Got BeaconPacket")
		spew.Dump(packet)

		if data, err := json.Marshal(packet); err == nil {
			_ = nc.Publish("beacon", data)
		}
	}).WithOnForwardData(func(packet ecla.ForwardDataPacket) {
		packet.Src = id

		fmt.Println("== [ECLA] Got ForwardDataPacket")
		spew.Dump(packet)

		if data, err := json.Marshal(packet); err == nil {
			_ = nc.Publish(packet.Dst, data)
		}
	})

	// On ForwardDatPacket
	_, _ = nc.Subscribe(id, func(msg *nats.Msg) {
		var fwd ecla.ForwardDataPacket
		if err := json.Unmarshal(msg.Data, &fwd); err == nil {
			fmt.Println("== [NATS] Got ForwardDataPacket")
			spew.Dump(fwd)

			ec.InsertForwardDataPacket(fwd)
		}
	})

	// On Beacon
	_, _ = nc.Subscribe("beacon", func(msg *nats.Msg) {
		var beacon ecla.BeaconPacket
		if err := json.Unmarshal(msg.Data, &beacon); err == nil {
			// Ignore own packet
			if beacon.Addr == id {
				return
			}

			fmt.Println("== [NATS] Got Beacon")
			spew.Dump(beacon)

			ec.InsertBeaconPacket(beacon)
		}
	})

	// Dial to ECLA
	go func() {
		if err := ec.Dial(os.Getenv("ECLA_BIND")); err != nil {
			log.Fatal(err)
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt

	ec.Close()
	nc.Close()
}
