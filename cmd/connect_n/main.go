package main

import (
	"flag"
	"github.com/BigJk/dtn7-ecla-go/pkg/ecla"
	"github.com/BigJk/dtn7-ecla-go/pkg/ecla/ws"
	"log"
	"os"
	"os/signal"
	"strings"
)

type arrayFlags []string

func (af *arrayFlags) String() string {
	return strings.Join(*af, ",")
}

func (af *arrayFlags) Set(value string) error {
	*af = append(*af, value)
	return nil
}

var addr arrayFlags

func main() {
	flag.Var(&addr, "addr", "specify which ECLA addresses to connect to")
	flag.Parse()

	if len(addr) == 0 {
		panic("no addresses given")
	}

	var eclaCons []*ws.ECLA
	for i := range addr {
		id := i
		eclaCons = append(eclaCons, ws.New("ConN", true).WithLogger(&ecla.LogLogger{}).WithReconnect(true).WithOnBeacon(func(packet ecla.BeaconPacket) {
			for i := range eclaCons {
				if id == i {
					continue
				}
				eclaCons[i].InsertBeaconPacket(packet)
			}
		}).WithOnForwardData(func(packet ecla.ForwardDataPacket) {
			for i := range eclaCons {
				if id == i {
					continue
				}
				eclaCons[i].InsertForwardDataPacket(packet)
			}
		}))
	}

	for i := range eclaCons {
		go func(i int) {
			if err := eclaCons[i].Dial(addr[i]); err != nil {
				panic(err)
			}
		}(i)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt

	log.Println("closing...")

	for i := range eclaCons {
		if err := eclaCons[i].Close(); err != nil {
			panic(err)
		}
	}
}
