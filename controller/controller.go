package controller

type Controller interface {
	Off()
	Fan()
	Cool()
	Heat()
}
