package main

import (
	"fmt"
	"net/http"
	"strconv"
)

func modeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" {
		m := r.FormValue("value")
		if m == "off" || m == "cool" || m == "heat" {
			ThermostatSetSystemMode(m)
		} else {
			http.Error(w, "", 400)
		}
	} else if r.Method != "GET" {
		http.Error(w, "", 405)
	}
	fmt.Fprintf(w, "{\"value\": \"%s\"}", ThermostatGetSystemMode())
}

func temperatureHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" {
		t, err := strconv.Atoi(r.FormValue("value"))
		if err == nil && t > 60 && t < 85 { // sanity range
			ThermostatSetDesiredTemperature(t)
		} else {
			http.Error(w, "", 400)
		}
	} else if r.Method != "GET" {
		http.Error(w, "", 405)

	}
	fmt.Fprintf(w, "{\"value\": \"%d\"}", ThermostatGetDesiredTemperature())
}

func ApiInit() {
	m := http.NewServeMux()
	m.HandleFunc("/mode", modeHandler)
	//m.HandleFunc("/fan", fanHandler)
	m.HandleFunc("/temperature", temperatureHandler)
	http.Handle("/", m)
	go http.ListenAndServe(":8000", m)
}
