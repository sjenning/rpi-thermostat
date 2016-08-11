package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/sjenning/rpi-thermostat/api"
	"github.com/sjenning/rpi-thermostat/controller/rpi"
	"github.com/sjenning/rpi-thermostat/sensor/sensortag"
	"github.com/sjenning/rpi-thermostat/thermostat"
)

func main() {
	sensor, err := sensortag.NewSensorTag()
	if err != nil {
		fmt.Printf("failed to create sensor: %v\n", err)
		return
	}

	controller, err := rpi.NewRpiController()
	if err != nil {
		fmt.Printf("failed to create controller: %v\n", err)
		return
	}

	thermostat, err := thermostat.NewThermostat(sensor, controller, 72)
	if err != nil {
		fmt.Printf("failed to create controller: %v\n", err)
		return
	}
	go thermostat.Run()

	apiserver := apiserver.NewAPIServer(thermostat)
	go apiserver.Run()

	done := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		close(done)
	}()
	<-done
	fmt.Printf("shutting down")
	controller.Off()
}
