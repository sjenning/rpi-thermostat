package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
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
	return (float32(rawdata)/4.0)*float32(0.03125)*(9.0/5.0) + 32
}

func handleTempNotification(c *gatt.Characteristic, b []byte, err error) {
	t := convertTemp(b[2:4])
	//fmt.Printf("%fF\n", t)
	ControllerReportTemperature(t)
}

func onPeriphConnected(p gatt.Peripheral, err error) {
	fmt.Println("Connected")
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		p.Device().CancelConnection(p)
	}()

	err = p.SetMTU(500)
	if err != nil {
		fmt.Printf("Failed to set MTU, err: %s\n", err)
	}

	uuid, err := gatt.ParseUUID("f000aa00-0451-4000-b000-000000000000")
	if err != nil {
		fmt.Printf("Failed to create UUID, err: %s\n", err)
		return
	}

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

	cs, err := p.DiscoverCharacteristics(nil, s)
	if err != nil {
		fmt.Printf("Failed to discover characteristics, err: %s\n", err)
		return
	}

	for _, c := range cs {
		_, err = p.DiscoverDescriptors(nil, c)
		if err != nil {
			fmt.Printf("Failed to discover descriptors, err: %s\n", err)
			return
		}
	}

	data := cs[0]
	config := cs[1]
	samplerate := cs[2]

	// Turn on notifications
	err = p.SetNotifyValue(data, handleTempNotification)
	if err != nil {
		fmt.Printf("Failed to write config characteristics, err: %s\n", err)
		return
	}

	// Set sample rate
	err = p.WriteCharacteristic(samplerate, []byte{0xFF}, false)
	if err != nil {
		fmt.Printf("Failed to write samplerate characteristics, err: %s\n", err)
		return
	}

	// Start sensors
	err = p.WriteCharacteristic(config, []byte{0x01}, false)
	if err != nil {
		fmt.Printf("Failed to write config characteristics, err: %s\n", err)
		return
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
