package apiserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/sjenning/rpi-thermostat/thermostat"
)

type apiserver struct {
	thermostat *thermostat.Thermostat
}

func (s *apiserver) updateHandler(r *http.Request) int {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusBadRequest
	}
	state := thermostat.ThermostatState{}
	err = json.Unmarshal(body, &state)
	if err != nil {
		return http.StatusBadRequest
	}
	s.thermostat.Put(state)
	return http.StatusOK
}

func (s *apiserver) apiHandler(w http.ResponseWriter, r *http.Request) {
	var code int

	switch r.Method {
	case "GET":
		code = http.StatusOK
	case "POST":
		code = s.updateHandler(r)
	default:
		code = http.StatusNotImplemented
	}
	response := s.thermostat.Get()
	fmt.Printf("%#v\n", response)
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

func NewAPIServer(thermostat *thermostat.Thermostat) *apiserver {
	return &apiserver{
		thermostat: thermostat,
	}
}

func (s *apiserver) Run() {
	http.HandleFunc("/api", s.apiHandler)
	http.Handle("/", http.FileServer(http.Dir("./ui")))
	http.ListenAndServe(":80", nil)
}
