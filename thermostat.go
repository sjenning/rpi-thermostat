package main

import (
	"fmt"
	"time"
)

var currentT = 75
var desiredT = 75
var fanmode = "auto"
var sysmode = "off"

func ThermostatSetFanMode(f string) {
	fanmode = f
}

func ThermostatSetSystemMode(m string) {
	sysmode = m
}

func ThermostatGetSystemMode() string {
	return sysmode
}

func ThermostatSetDesiredTemperature(t int) {
	desiredT = t
}

func ThermostatGetDesiredTemperature() int {
	return desiredT
}

func ThermostatReportCurrentTemperature(t int) {
	currentT = t
}

func ThermostatMainLoop() {
	for {
		<-time.After(time.Second)
		fmt.Printf("desired %dF, current %dF, sysmode %s, fanmode %s\n", desiredT, currentT, sysmode, fanmode)
		switch sysmode {
		case "off":
			StopHeat()
			StopCool()
			if fanmode == "auto" {
				StopFan()
			}
		case "cool":
			if currentT > desiredT {
				StartFan()
				StartCool()
			} else if currentT < desiredT {
				if fanmode == "auto" {
					StopFan()
				}
				StopCool()
			}
		case "heat":
			if currentT < desiredT {
				StartFan()
				StartHeat()
			} else if currentT > desiredT {
				if fanmode == "auto" {
					StopFan()
				}
				StopHeat()
			}
		}
	}
}

func ThermostatInit() {
	go ThermostatMainLoop()
}
