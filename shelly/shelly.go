package shelly

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"barista.run/bar"
	"barista.run/base/value"
	l "barista.run/logging"
	"barista.run/outputs"
	"barista.run/timing"
)

// endpoint: http://address/status
// response: status.json
type Shelly1Status struct {
	WifiSta struct {
		Connected bool   `json:"connected"`
		Ssid      string `json:"ssid"`
		IP        string `json:"ip"`
		Rssi      int    `json:"rssi"`
	} `json:"wifi_sta"`
	Cloud struct {
		Enabled   bool `json:"enabled"`
		Connected bool `json:"connected"`
	} `json:"cloud"`
	Mqtt struct {
		Connected bool `json:"connected"`
	} `json:"mqtt"`
	Time          string `json:"time"`
	Unixtime      int    `json:"unixtime"`
	Serial        int    `json:"serial"`
	HasUpdate     bool   `json:"has_update"`
	Mac           string `json:"mac"`
	CfgChangedCnt int    `json:"cfg_changed_cnt"`
	ActionsStats  struct {
		Skipped int `json:"skipped"`
	} `json:"actions_stats"`
	Relays []struct {
		Ison           bool   `json:"ison"`
		HasTimer       bool   `json:"has_timer"`
		TimerStarted   int    `json:"timer_started"`
		TimerDuration  int    `json:"timer_duration"`
		TimerRemaining int    `json:"timer_remaining"`
		Source         string `json:"source"`
	} `json:"relays"`
	Meters []struct {
		Power   float64 `json:"power"`
		IsValid bool    `json:"is_valid"`
	} `json:"meters"`
	Inputs []struct {
		Input    int    `json:"input"`
		Event    string `json:"event"`
		EventCnt int    `json:"event_cnt"`
	} `json:"inputs"`
	ExtSensors struct {
	} `json:"ext_sensors"`
	ExtTemperature struct {
	} `json:"ext_temperature"`
	ExtHumidity struct {
	} `json:"ext_humidity"`
	Update struct {
		Status     string `json:"status"`
		HasUpdate  bool   `json:"has_update"`
		NewVersion string `json:"new_version"`
		OldVersion string `json:"old_version"`
	} `json:"update"`
	RAMTotal int `json:"ram_total"`
	RAMFree  int `json:"ram_free"`
	FsSize   int `json:"fs_size"`
	FsFree   int `json:"fs_free"`
	Uptime   int `json:"uptime"`
}

// endpoint: http://address/relay/0?turn=toggle
// response: toggle.json
type Shelly1ToggleResponse struct {
	Ison           bool   `json:"ison,omitempty"`
	HasTimer       bool   `json:"has_timer,omitempty"`
	TimerStarted   int    `json:"timer_started,omitempty"`
	TimerDuration  int    `json:"timer_duration,omitempty"`
	TimerRemaining int    `json:"timer_remaining,omitempty"`
	Source         string `json:"source,omitempty"`
}

const (
	shelly1toggle string = "/relay/0?turn=toggle"
	shelly1status string = "/status"
)

type ShellyState struct {
	IsOn            bool
	Address         string
	UpdateAvailable bool
	UpdateVersion   string
	RamTotal        int
	RamFree         int
	FsSize          int
	FsFree          int
}

func (s ShellyState) Connected() bool {
	return s.IsOn
}

func (s ShellyState) Toggle() {
	toggleShelly(s.Address)
}

func (s ShellyState) IsUpdateAvailable() bool {
	return s.UpdateAvailable
}

func (s ShellyState) GetVersion() string {
	return s.UpdateVersion
}

func (s ShellyState) DiskUtilization() float64 {
	return (float64(s.FsSize) - float64(s.FsFree)) / float64(s.FsSize) * 100
}

func (s ShellyState) MemoryUtilization() float64 {
	return (float64(s.RamTotal) - float64(s.RamFree)) / float64(s.RamTotal) * 100
}

type Module struct {
	addr       string
	scheduler  *timing.Scheduler
	outputFunc value.Value
}

func New(address string) *Module {

	m := &Module{
		addr:      address,
		scheduler: timing.NewScheduler(),
	}
	l.Label(m, address)
	l.Register(m, "outputFunc")
	m.RefreshInterval(5 * time.Minute)

	m.Output(func(s ShellyState) bar.Output {
		if s.Connected() {
			return outputs.Text("SHELLY")
		}
		return nil
	})

	return m
}

func (m *Module) Output(outputFunc func(ShellyState) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	state := getShellyStatus(m.addr)

	outputFunc := m.outputFunc.Get().(func(ShellyState) bar.Output)
	nextOutputFunc, done := m.outputFunc.Subscribe()

	defer done()

	for {
		s.Output(outputFunc(state))
		select {
		case <-m.scheduler.C:
			state = getShellyStatus(m.addr)
		case <-nextOutputFunc:
			outputFunc = m.outputFunc.Get().(func(ShellyState) bar.Output)
		}

	}
}

func getShellyStatus(address string) ShellyState {

	var shellyState ShellyState

	shellyState.Address = address

	resp, err := http.Get("http://" + address + shelly1status)
	if err != nil {
		shellyState.IsOn = false
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		shellyState.IsOn = false
	}

	var statusResponse Shelly1Status
	if err := json.Unmarshal(body, &statusResponse); err != nil {
		shellyState.IsOn = false
	}
	shellyState.IsOn = statusResponse.Relays[0].Ison

	// stats
	shellyState.UpdateAvailable = statusResponse.HasUpdate
	shellyState.UpdateVersion = statusResponse.Update.NewVersion

	shellyState.FsFree = statusResponse.FsFree
	shellyState.FsSize = statusResponse.FsSize
	shellyState.RamFree = statusResponse.RAMFree
	shellyState.RamTotal = statusResponse.RAMTotal

	return shellyState
}

func toggleShelly(address string) ShellyState {

	var shellyToggle Shelly1ToggleResponse
	var shellyState ShellyState

	resp, err := http.Get("http://" + address + shelly1toggle)
	if err != nil {
		shellyToggle.Ison = false
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		shellyToggle.Ison = false
	}

	if err := json.Unmarshal(body, &shellyToggle); err != nil {
		shellyToggle.Ison = false
	}

	shellyState.IsOn = shellyToggle.Ison

	return shellyState

}
