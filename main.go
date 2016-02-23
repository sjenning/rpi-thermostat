package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/linux/cmd"
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

// GPIO

func Start(p rpio.Pin) {
	p.Low()
}

func Stop(p rpio.Pin) {
	p.High()
}

// Sensor

func onStateChanged(d gatt.Device, s gatt.State) {
	log.Println("State:", s)
	switch s {
	case gatt.StatePoweredOn:
		log.Println("Scanning...")
		d.Scan([]gatt.UUID{}, false)
		return
	default:
		d.StopScanning()
	}
}

func onPeriphDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	if !strings.Contains(a.LocalName, "SensorTag") {
		return
	}

	// Stop scanning once we've got the peripheral we're looking for.
	p.Device().StopScanning()

	log.Printf("\nPeripheral ID:%s, NAME:(%s)\n", p.ID(), p.Name())
	log.Println("  Local Name        =", a.LocalName)
	log.Println("  TX Power Level    =", a.TxPowerLevel)
	log.Println("  Manufacturer Data =", a.ManufacturerData)
	log.Println("  Service Data      =", a.ServiceData)
	log.Println("")

	p.Device().Connect(p)
}

func convertTemp(b []byte) int {
	rawdata := binary.LittleEndian.Uint16(b)
	return int((float32(rawdata)/4.0)*float32(0.03125)*(9.0/5.0) + 32)
}

func handleTempNotification(c *gatt.Characteristic, b []byte, err error) {
	t := convertTemp(b[2:4])
	if t > 50 && t < 90 { // sanity range
		current = t
		updateState()
	}
}

func onPeriphConnected(p gatt.Peripheral, err error) {
	log.Println("Connected")
	peripheral = p

	err = p.SetMTU(500)
	if err != nil {
		log.Printf("Failed to set MTU, err: %s\n", err)
	}

	uuid, err := gatt.ParseUUID("f000aa00-0451-4000-b000-000000000000")
	if err != nil {
		log.Printf("Failed to create UUID, err: %s\n", err)
		return
	}

	ss, err := p.DiscoverServices(nil)
	if err != nil {
		log.Printf("Failed to discover IR sensor service, err: %s\n", err)
		return
	}

	var s *gatt.Service
	for _, s = range ss {
		if s.UUID().Equal(uuid) {
			break
		}
	}

	cs, err := p.DiscoverCharacteristics(nil, s)
	if err != nil {
		log.Printf("Failed to discover characteristics, err: %s\n", err)
		return
	}

	for _, c := range cs {
		_, err = p.DiscoverDescriptors(nil, c)
		if err != nil {
			log.Printf("Failed to discover descriptors, err: %s\n", err)
			return
		}
	}

	data := cs[0]
	config := cs[1]
	samplerate := cs[2]

	// Turn on notifications
	err = p.SetNotifyValue(data, handleTempNotification)
	if err != nil {
		log.Printf("Failed to write config characteristics, err: %s\n", err)
		return
	}

	// Set sample rate
	err = p.WriteCharacteristic(samplerate, []byte{0xFF}, false)
	if err != nil {
		log.Printf("Failed to write samplerate characteristics, err: %s\n", err)
		return
	}

	// Start sensors
	err = p.WriteCharacteristic(config, []byte{0x01}, false)
	if err != nil {
		log.Printf("Failed to write config characteristics, err: %s\n", err)
		return
	}
}

var done = make(chan bool)

func onPeriphDisconnected(p gatt.Peripheral, err error) {
	if exiting {
		disconnected <- true
	} else {
		done <- true
		peripheral = nil
	}
}

func updateState() {
	if current == 0 {
		Stop(fan)
		Stop(cool)
		Stop(heat)
		return
	}
	switch sysmode {
	case "off":
		Stop(heat)
		Stop(cool)
		if fanmode == "auto" {
			Stop(fan)
		}
	case "cool":
		if current > desired {
			Start(fan)
			Start(cool)
		} else if current < desired {
			if fanmode == "auto" {
				Stop(fan)
			}
			Stop(cool)
		}
		Stop(heat)
	case "heat":
		if current < desired {
			Start(fan)
			Start(heat)
		} else if current > desired {
			if fanmode == "auto" {
				Stop(fan)
			}
			Stop(heat)
		}
		Stop(cool)
	}

	if fanmode == "on" {
		Start(fan)
	}
	go func() {
		led := rpio.Pin(16)
		led.Output()
		led.Low()
		<-time.After(100 * time.Millisecond)
		led.High()
	}()
}

func readSettingsFile() error {
	settingsFile, err := os.Open("settings.json")
	if err != nil {
		return fmt.Errorf("settings.json file not found, using defaults")
	}

	var settings Thermostat
	if err = json.NewDecoder(settingsFile).Decode(&settings); err != nil {
		return err
	}

	desired = settings.Desired
	fanmode = settings.Fanmode
	sysmode = settings.Sysmode
	settingsFile.Close()
	return nil
}

func writeSettingsFile() error {
	settingsFile, err := os.Create("settings.json")
	if err != nil {
		return err
	}
	defer settingsFile.Close()

	var settings Thermostat
	settings.Desired = desired
	settings.Fanmode = fanmode
	settings.Sysmode = sysmode
	output, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = settingsFile.Write(output)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	err := readSettingsFile()
	if err != nil {
		log.Printf("WARNING: %s", err)
	}
	defer writeSettingsFile()

	// Sensor
	d, err := gatt.NewDevice(DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	d.Handle(
		gatt.PeripheralDiscovered(onPeriphDiscovered),
		gatt.PeripheralConnected(onPeriphConnected),
		gatt.PeripheralDisconnected(onPeriphDisconnected),
	)

	d.Init(onStateChanged)

	// API
	http.HandleFunc("/api", apiHandler)
	http.Handle("/", http.FileServer(http.Dir("./ui")))
	go http.ListenAndServe(":80", nil)

	// GPIO
	err = rpio.Open()
	if err != nil {
		log.Printf("Failed to open controller, err: %s\n", err)
		return
	}
	defer func() {
		for _, p := range []rpio.Pin{fan, cool, heat} {
			Stop(p)
		}
		rpio.Close()
	}()

	for _, p := range []rpio.Pin{fan, cool, heat} {
		p.Output()
		Stop(p)
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		done <- true
	}()
	<-done

	exiting = true
	Stop(fan)
	Stop(cool)
	Stop(heat)
	if peripheral != nil {
		d.CancelConnection(peripheral)
		<-disconnected
		return
	}
	os.Exit(1)
}
