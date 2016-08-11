package main

import (
	"fmt"

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

	apiserver := apiserver.NewAPIServer(thermostat)
	apiserver.Run()
}
