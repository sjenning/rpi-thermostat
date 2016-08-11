package thermostat

import (
	"fmt"
	"time"

	"github.com/sjenning/rpi-thermostat/controller"
	"github.com/sjenning/rpi-thermostat/sensor"
)

type Thermostat struct {
	sensor     sensor.Sensor
	controller controller.Controller
	sysmode    string
	fanmode    string
	desired    int
	current    int
}

type ThermostatState struct {
	Current int    `json:"current"`
	Desired int    `json:"desired"`
	Sysmode string `json:"sysmode"`
	Fanmode string `json:"fanmode"`
}

func NewThermostat(sensor sensor.Sensor, controller controller.Controller, desired int) (*Thermostat, error) {
	return &Thermostat{
		sensor:     sensor,
		controller: controller,
		desired:    desired,
		current:    0,
	}, nil
}

func (t *Thermostat) Run() {
	for {
		current, err := t.sensor.GetTemperature()
		if err != nil {
			fmt.Println(err)
			fmt.Printf("switching system off due to sensor failure")
			t.controller.Off()
			t.sysmode = "off"
		} else {
			t.current = current
			t.Update()
		}
		<-time.After(time.Minute)
	}
}

func (t *Thermostat) Put(state ThermostatState) {
	t.current = state.Current
	t.desired = state.Desired
	t.sysmode = state.Sysmode
	t.fanmode = state.Fanmode
	t.Update()
}

func (t *Thermostat) Get() ThermostatState {
	return ThermostatState{
		Current: t.current,
		Desired: t.desired,
		Sysmode: t.sysmode,
		Fanmode: t.fanmode,
	}
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
