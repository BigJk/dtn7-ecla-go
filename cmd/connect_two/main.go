package main

import (
	"github.com/BigJk/dtn7-ecla-go/pkg/ecla"
	"log"
	"os"
	"os/signal"

	"github.com/BigJk/dtn7-ecla-go/pkg/ecla/ws"
)

func main() {
	conA := ws.New("2DirectConn", true).WithLogger(&ecla.LogLogger{}).WithReconnect(true)
	conB := ws.New("2DirectConn", true).WithLogger(&ecla.LogLogger{}).WithReconnect(true)

	conA.WithOnIdRequest(func() string {
		return "conA"
	})
	conA.WithOnRegistered(func(packet ecla.RegisteredPacket) {
		log.Println("[CONA] Registerd to", packet.NodeID)
	})
	conA.WithOnBeacon(func(packet ecla.BeaconPacket) {
		log.Println("[CONA] BEACON")
		conB.InsertBeaconPacket(packet)
	})
	conA.WithOnForwardData(func(packet ecla.ForwardDataPacket) {
		log.Println("[CONA] FWD ->", packet.Dst)
		if packet.Dst == "conB" {
			conB.InsertForwardDataPacket(packet)
		}
	})

	conB.WithOnIdRequest(func() string {
		return "conB"
	})
	conB.WithOnRegistered(func(packet ecla.RegisteredPacket) {
		log.Println("=== [CONB] Registerd to", packet.NodeID)
	})
	conB.WithOnBeacon(func(packet ecla.BeaconPacket) {
		log.Println("=== [CONB] BEACON")
		conA.InsertBeaconPacket(packet)
	})
	conB.WithOnForwardData(func(packet ecla.ForwardDataPacket) {
		log.Println("=== [CONB] FWD ->", packet.Dst)
		if packet.Dst == "conA" {
			conA.InsertForwardDataPacket(packet)
		}
	})

	go func() {
		if err := conA.Dial("127.0.0.1:3000"); err != nil {
			log.Fatal(err)
		}
		log.Println("conA done")
	}()

	go func() {
		if err := conB.Dial("127.0.0.1:3001"); err != nil {
			log.Fatal(err)
		}
		log.Println("conB done")
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt

	log.Println("closing...")

	_ = conA.Close()
	_ = conB.Close()
}
