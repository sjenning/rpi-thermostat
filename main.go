package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
)

var done = make(chan bool)

func exit() {
	TemperatureExit()
	ControllerExit()
	done <- true
}

func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatalf("usage: %s [options] peripheral-id\n", os.Args[0])
	}

	id := strings.ToUpper(flag.Args()[0])
	TemperatureInit(id)
	ControllerInit()
	ThermostatInit()
	ApiInit()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		exit()
	}()

	<-done
	fmt.Println("Done")
}
