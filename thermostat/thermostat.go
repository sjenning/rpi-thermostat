package thermostat

import (
	"fmt"
	"time"

	"github.com/spartacus06/rpi-thermostat/controller"
	"github.com/spartacus06/rpi-thermostat/sensor"
)

type Thermostat struct {
	sensor     sensor.Sensor
	controller controller.Controller
	sysmode    string
	fanmode    string
	desired    int
	current    int
}

func NewThermostat(sensor sensor.Sensor, controller controller.Controller, desired int) (*Thermostat, error) {
	return &Thermostat{
		sensor:     sensor,
		controller: controller,
		desired:    desired,
		current:    0,
	}, nil
}

func (t *Thermostat) Run(done chan struct{}) {
	for {
		select {
		case <-time.After(time.Minute):
		case <-done:
			return
		}
		current, err := t.sensor.GetTemperature()
		if err != nil {
			fmt.Println(err)
			fmt.Printf("switching system off due to sensor failure")
			t.controller.Off()
			t.sysmode = "off"
			continue
		}
		t.current = current
		t.Update()
	}
}

func (t *Thermostat) SetSysMode(mode string) {
	t.sysmode = mode
	t.Update()
}

func (t *Thermostat) SetFanMode(mode string) {
	t.fanmode = mode
	t.Update()
}

func (t *Thermostat) SetDesired(desired int) {
	t.desired = desired
	t.Update()
}

func (t *Thermostat) Update() {
	if t.sysmode == "cool" && t.current > t.desired {
		t.controller.Cool()
		return
	}

	if t.sysmode == "heat" && t.current < t.desired {
		t.controller.Heat()
		return
	}

	if t.fanmode == "on" {
		t.controller.Fan()
		return
	}

	t.controller.Off()
}
