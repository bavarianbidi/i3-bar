package toggle

import (
	"os/exec"
	"strings"
	"time"

	"barista.run/bar"
	"barista.run/colors"
	"barista.run/outputs"
	"barista.run/pango"
)

type SystemdModule struct {
	*Module
	serviceName string
}

func NewSystemdUserService(serviceName string) *SystemdModule {
	m := &SystemdModule{
		serviceName: serviceName,
	}
	m.Module = New(
		m.toggleSystemdUserService,
		m.clickSystemdUserService,
		m.outputSystemdUserService,
		time.Second*5,
	)
	return m
}

func (m *SystemdModule) toggleSystemdUserService() string {
	out, _ := exec.Command("systemctl", "--user", "is-active", m.serviceName).Output()
	return strings.TrimSpace(string(out))
}

func (m *SystemdModule) outputSystemdUserService(serviceState string) *bar.Segment {
	var stateColor string
	switch serviceState {
	case "active":
		stateColor = "#238555"
	case "inactive":
		stateColor = "#972822"
	default:
		stateColor = "#f70"
	}
	return outputs.Pango(
		pango.Icon("mdi-arrow-decision").Alpha(0.6),
		pango.Text(m.serviceName).Color(colors.Hex(stateColor)),
	)
}

func (m *SystemdModule) clickSystemdUserService() {
	out, _ := exec.Command("systemctl", "--user", "is-active", m.serviceName).Output()
	var toggleCmd string

	switch strings.TrimSpace(string(out)) {
	case "active":
		toggleCmd = "stop"
	case "inactive":
		toggleCmd = "start"
	default:
		toggleCmd = "restart"
	}
	_, _ = exec.Command("systemctl", "--user", toggleCmd, m.serviceName).CombinedOutput()

	m.refresh()
}
