package sensortag

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/paypal/gatt"
	"github.com/sjenning/rpi-thermostat/sensor"
)

type sensorTag struct {
	peripheral gatt.Peripheral
	device     gatt.Device
	service    *gatt.Service
	data       *gatt.Characteristic
	config     *gatt.Characteristic
}

func (s *sensorTag) onPeriphDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	fmt.Printf("Advertisement from %s\n", a.LocalName)
	if !strings.Contains(a.LocalName, "SensorTag") {
		return
	}
	p.Device().StopScanning()
	p.Device().Connect(p)
}

func (s *sensorTag) onPeriphConnected(p gatt.Peripheral, err error) {
	fmt.Printf("Connected to %s\n", p.Name())
	s.peripheral = p

	err = p.SetMTU(500)
	if err != nil {
		fmt.Printf("failed to set MTU: %s\n", err)
	}

	services, err := p.DiscoverServices(nil)
	if err != nil {
		fmt.Printf("failed to discover sensor services: %s\n", err)
		return
	}

	uuid, err := gatt.ParseUUID("f000aa00-0451-4000-b000-000000000000")
	if err != nil {
		fmt.Printf("failed to create UUID: %s\n", err)
		return
	}

	for _, service := range services {
		if service.UUID().Equal(uuid) {
			s.service = service
			break
		}
	}

	characteristics, err := p.DiscoverCharacteristics(nil, s.service)
	if err != nil {
		fmt.Printf("failed to discover characteristics, err: %s\n", err)
		return
	}

	s.data = characteristics[0]
	s.config = characteristics[1]
}

func (s *sensorTag) onPeriphDisconnected(p gatt.Peripheral, err error) {
	fmt.Printf("peripheral disconnected, starting scan")
	s.peripheral = nil
	p.Device().Scan([]gatt.UUID{}, false)
}

func onStateChanged(d gatt.Device, s gatt.State) {

}

func NewSensorTag() (sensor.Sensor, error) {
	DefaultClientOptions := []gatt.Option{
		gatt.LnxMaxConnections(1),
		gatt.LnxDeviceID(-1, true),
	}

	device, err := gatt.NewDevice(DefaultClientOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to open device: %s\n", err)
	}

	s := &sensorTag{device: device}
	device.Handle(
		gatt.PeripheralDiscovered(s.onPeriphDiscovered),
		gatt.PeripheralConnected(s.onPeriphConnected),
		gatt.PeripheralDisconnected(s.onPeriphDisconnected),
	)

	device.Init(onStateChanged)
	return s, nil
}

func (s *sensorTag) GetTemperature() (int, error) {
	if s.peripheral == nil {
		return 0, fmt.Errorf("sensor not connected")
	}
	err := s.peripheral.WriteCharacteristic(s.config, []byte{0x01}, false)
	if err != nil {
		return 0, fmt.Errorf("failed to enable sensor: %s\n", err)
	}

	bytes, err := s.peripheral.ReadCharacteristic(s.data)
	if err != nil {
		return 0, fmt.Errorf("failed to read from sensor: %s\n", err)
	}

	return int((float32(binary.LittleEndian.Uint16(bytes[2:4]))/4.0)*float32(0.03125)*(9.0/5.0) + 32), nil
}
