package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
	"github.com/stianeikeland/go-rpio"
)

var done = make(chan struct{})

func onStateChanged(d gatt.Device, s gatt.State) {
	fmt.Println("State:", s)
	switch s {
	case gatt.StatePoweredOn:
		fmt.Println("Scanning...")
		d.Scan([]gatt.UUID{}, false)
		return
	default:
		d.StopScanning()
	}
}

func onPeriphDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	id := strings.ToUpper(flag.Args()[0])
	if strings.ToUpper(p.ID()) != id {
		return
	}

	// Stop scanning once we've got the peripheral we're looking for.
	p.Device().StopScanning()

	fmt.Printf("\nPeripheral ID:%s, NAME:(%s)\n", p.ID(), p.Name())
	fmt.Println("  Local Name        =", a.LocalName)
	fmt.Println("  TX Power Level    =", a.TxPowerLevel)
	fmt.Println("  Manufacturer Data =", a.ManufacturerData)
	fmt.Println("  Service Data      =", a.ServiceData)
	fmt.Println("")

	p.Device().Connect(p)
}

func convertTemp(b []byte) float32 {
	rawdata := binary.LittleEndian.Uint16(b)
	//fmt.Printf("rawdata %d\n", rawdata)
	return (float32(rawdata)/4.0)*float32(0.03125)*(9.0/5.0) + 32
}

func onPeriphConnected(p gatt.Peripheral, err error) {
	fmt.Println("Connected")
	defer p.Device().CancelConnection(p)

	err = p.SetMTU(500)
	if err != nil {
		fmt.Printf("Failed to set MTU, err: %s\n", err)
	}

	uuid, err := gatt.ParseUUID("f000aa00-0451-4000-b000-000000000000")
	if err != nil {
		fmt.Printf("Failed to create UUID, err: %s\n", err)
		return
	}

	fmt.Printf("UUID: %s\n", uuid.String())

	ss, err := p.DiscoverServices(nil)
	if err != nil {
		fmt.Printf("Failed to discover IR sensor service, err: %s\n", err)
		return
	}

	var s *gatt.Service
	for _, s = range ss {
		if s.UUID().Equal(uuid) {
			break
		}
	}

	fmt.Printf("%s\n", s.UUID().String())

	cs, err := p.DiscoverCharacteristics(nil, s)
	if err != nil {
		fmt.Printf("Failed to discover characteristics, err: %s\n", err)
		return
	}

	data := cs[0]
	config := cs[1]
	samplerate := cs[2]

	err = p.WriteCharacteristic(samplerate, []byte{0x0A}, false)
	if err != nil {
		fmt.Printf("Failed to write samplerate characteristics, err: %s\n", err)
		return
	}

	err = p.WriteCharacteristic(config, []byte{0x01}, false)
	if err != nil {
		fmt.Printf("Failed to write config characteristics, err: %s\n", err)
		return
	}

	if err = rpio.Open(); err != nil {
		fmt.Printf("Failed to open GPIO, err: %s\n", err)
		return
	}

	defer rpio.Close()

	pins := make([]rpio.Pin, 5)
	pins[0] = rpio.Pin(17)
	pins[1] = rpio.Pin(21)
	pins[2] = rpio.Pin(22)
	pins[3] = rpio.Pin(23)
	pins[4] = rpio.Pin(24)

	for _, l := range pins {
		l.Output()
	}

	levels := []float32{0.0, 68.0, 72.0, 76.0, 80.0}

	for i := 0; i < 300; i++ {
		bytes, err := p.ReadCharacteristic(data)
		if err != nil {
			fmt.Printf("Failed to read data characteristics, err: %s\n", err)
		}

		//fmt.Printf("Data: %x\n", bytes)
		ambtemp := convertTemp(bytes[2:4])
		fmt.Println("AMB", ambtemp)
		objtemp := convertTemp(bytes[0:2])
		fmt.Println("OBJ", objtemp)
		fmt.Println()

		for i, l := range levels {
			if objtemp > l {
				pins[i].High()
			} else {
				pins[i].Low()
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	for _, l := range pins {
		l.Low()
	}
}

func onPeriphDisconnected(p gatt.Peripheral, err error) {
	fmt.Println("Disconnected")
	close(done)
}

func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatalf("usage: %s [options] peripheral-id\n", os.Args[0])
	}

	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	// Register handlers.
	d.Handle(
		gatt.PeripheralDiscovered(onPeriphDiscovered),
		gatt.PeripheralConnected(onPeriphConnected),
		gatt.PeripheralDisconnected(onPeriphDisconnected),
	)

	d.Init(onStateChanged)
	<-done
	fmt.Println("Done")
}
