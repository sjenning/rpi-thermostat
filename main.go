package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/linux/cmd"
	"github.com/spartacus06/rpi-thermostat/controller/rpi"
	"github.com/spartacus06/rpi-thermostat/sensor/sensortag"
	"github.com/spartacus06/rpi-thermostat/thermostat"
	"github.com/stianeikeland/go-rpio"
)

var (
	current = 0
	desired = 75
	fanmode = "auto"
	sysmode = "off"

	fan  = rpio.Pin(17)
	cool = rpio.Pin(21)
	heat = rpio.Pin(22)

	exiting      = false
	disconnected = make(chan bool)
	peripheral   gatt.Peripheral

	DefaultClientOptions = []gatt.Option{
		gatt.LnxMaxConnections(1),
		gatt.LnxDeviceID(-1, true),
	}

	DefaultServerOptions = []gatt.Option{
		gatt.LnxMaxConnections(1),
		gatt.LnxDeviceID(-1, true),
		gatt.LnxSetAdvertisingParameters(&cmd.LESetAdvertisingParameters{
			AdvertisingIntervalMin: 0x00f4,
			AdvertisingIntervalMax: 0x00f4,
			AdvertisingChannelMap:  0x7,
		}),
	}
)

// API

type Thermostat struct {
	Current int    `json:"current"`
	Desired int    `json:"desired"`
	Sysmode string `json:"sysmode"`
	Fanmode string `json:"fanmode"`
}

func updateHandler(r *http.Request) int {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusBadRequest
	}

	var t Thermostat
	err = json.Unmarshal(body, &t)
	if err != nil {
		return http.StatusBadRequest
	}

	if t.Desired >= 60 && t.Desired <= 85 {
		desired = t.Desired
	}
	if t.Current >= 50 && t.Current <= 90 {
		current = t.Current
	}
	sysmode = t.Sysmode
	fanmode = t.Fanmode
	updateState()
	return http.StatusOK
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	var code int

	switch r.Method {
	case "GET":
		code = http.StatusOK
	case "POST":
		code = updateHandler(r)
	default:
		code = http.StatusNotImplemented
	}
	log.Printf("current: %d, desired: %d, sysmode: %s, fanmode: %s\n", current, desired, sysmode, fanmode)
	response := Thermostat{current, desired, sysmode, fanmode}
	json, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error marshalling JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if code != 200 {
		http.Error(w, "", code)
	}
	w.Write(json)
	log.Printf("%s %s %s %d", r.RemoteAddr, r.Method, r.URL, 200)
}

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

	thermostat, err := thermostat.NewThermostat(sensor, controller, done)
	if err != nil {
		fmt.Printf("failed to create controller: %v\n", err)
		return
	}

	// API
	http.HandleFunc("/api", apiHandler)
	http.Handle("/", http.FileServer(http.Dir("./ui")))

	done := make(chan struct{})
	go thermostat.Run(done)
	http.ListenAndServe(":80", nil)
	close(done)
}
