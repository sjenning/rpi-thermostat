package rpi

import (
	"fmt"

	"github.com/spartacus06/rpi-thermostat/controller"
	"github.com/stianeikeland/go-rpio"
)

type rpiController struct{}

var (
	fan  = rpio.Pin(17)
	cool = rpio.Pin(21)
	heat = rpio.Pin(22)
)

func NewRpiController() (controller.Controller, error) {
	err := rpio.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open RPi controller: %s\n", err)
	}
	return &rpiController{}, nil
}

func (c *rpiController) Off() {
	cool.High()
	heat.High()
	fan.High()
}

func (c *rpiController) Fan() {
	cool.High()
	heat.High()
	fan.Low()
}

func (c *rpiController) Cool() {
	cool.Low()
	heat.High()
	fan.Low()
}

func (c *rpiController) Heat() {
	cool.High()
	heat.Low()
	fan.Low()
}
