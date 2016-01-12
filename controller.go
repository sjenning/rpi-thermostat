package main

import (
	"fmt"

	"github.com/stianeikeland/go-rpio"
)

var fan = rpio.Pin(17)
var cool = rpio.Pin(21)
var heat = rpio.Pin(22)
var pins = []rpio.Pin{fan, cool, heat}

func ControllerInit() error {
	err := rpio.Open()
	if err != nil {
		fmt.Printf("Failed to open controller, err: %s\n", err)
		return err
	}

	for _, p := range pins {
		p.Output()
		p.High()
	}

	return nil
}

func ControllerExit() {
	StopAll()
	rpio.Close()
}

func StopAll() {
	for _, p := range pins {
		p.High()
	}
}

func StartFan() {
	fan.Low()
}

func StartCool() {
	cool.Low()
}

func StartHeat() {
	heat.Low()
}

func StopFan() {
	fan.High()
}

func StopCool() {
	cool.High()
}

func StopHeat() {
	heat.High()
}
