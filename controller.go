package main

import (
	"fmt"

	"github.com/stianeikeland/go-rpio"
)

var fan = rpio.Pin(17)
var cool = rpio.Pin(21)
var heat = rpio.Pin(22)

func ControllerOpen() error {
	err := rpio.Open()
	if err != nil {
		fmt.Printf("Failed to open controller, err: %s\n", err)
		return err
	}
	return nil
}

func ControllerClose() {
	rpio.Close()
}

func ControllerReportTemperature(t float32) {
	fmt.Printf("Reported temperature: %fF\n", t)
}
