package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/BigJk/dtn7-ecla-go/pkg/ecla"
)

func main() {
	conA := ecla.New("2DirectConn")
	conB := ecla.New("2DirectConn")

	conA.SetOnBeacon(func(packet ecla.BeaconPacket) {
		packet.Addr = "conA"
		conB.InsertBeaconPacket(packet)
	}).SetOnForwardData(func(packet ecla.ForwardDataPacket) {
		packet.From = "conA"
		if packet.To == "conB" {
			conB.InsertForwardDataPacket(packet)
		}
	})

	conB.SetOnBeacon(func(packet ecla.BeaconPacket) {
		packet.Addr = "conB"
		conA.InsertBeaconPacket(packet)
	}).SetOnForwardData(func(packet ecla.ForwardDataPacket) {
		packet.From = "conB"
		if packet.To == "conA" {
			conA.InsertForwardDataPacket(packet)
		}
	})

	if err := conA.Dial("127.0.0.1:8172"); err != nil {
		log.Fatal(err)
	}

	if err := conB.Dial("127.0.0.1:8173"); err != nil {
		log.Fatal(err)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt

	log.Println("closing...")

	conA.Close()
	conB.Close()
}
