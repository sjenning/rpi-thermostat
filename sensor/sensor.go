package sensor

type Sensor interface {
	GetTemperature() (int, error)
}
