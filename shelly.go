package main

import (
	"github.com/barista-run/barista/bar"
	"github.com/barista-run/barista/base/click"
	"github.com/barista-run/barista/colors"
	"github.com/barista-run/barista/modules/meta/split"
	"github.com/barista-run/barista/outputs"
	"github.com/barista-run/barista/pango"
	"github.com/bavarianbidi/i3-bar/shelly"
)

func shellyStatus(address, icon string) (bar.Module, bar.Module) {
	return split.New(shelly.New(address).
		//RefreshInterval(3*time.Second).
		Output(func(s shelly.ShellyState) bar.Output {

			color := colorOn
			iconAppendix := ""

			out := outputs.Group()

			if s.Reachable() {
				if !s.Connected() {
					color = colorOff
					iconAppendix = "-outline"
				}
				out.Append(
					outputs.Pango(
						pango.Icon("mdi-" + icon + iconAppendix).Color(colors.Hex(color)),
					))

				out.OnClick(click.Left(func() {
					s.Toggle()
				}))

				if s.IsUpdateAvailable() {
					out.Append(outputs.Pango(
						pango.Icon("mdi-package-down").Color(colors.Hex("#34eb55")),
						spacer,
						pango.Textf("version %s available", s.GetVersion()),
					))
				}
				if !s.IsUpdateAvailable() {
					out.Append(outputs.Pango(
						pango.Icon("mdi-package-down"),
						spacer,
						pango.Textf("up to date"),
					))
				}

				out.Append(outputs.Pango(
					pango.Icon("mdi-harddisk"),
					spacer,
					pango.Textf("%.0f%% used", s.DiskUtilization()),
				))

				out.Append(outputs.Pango(
					pango.Icon("mdi-memory"),
					spacer,
					pango.Textf("%.0f%% RAM usage", s.MemoryUtilization()),
				))
			} else {

				color = colorOff
				iconAppendix = "-off"

				out.Append(
					outputs.Pango(
						pango.Icon("mdi-" + icon + iconAppendix).Color(colors.Hex(color)),
					))

				out.Append(outputs.Pango(
					spacer,
					pango.Textf("shelly not reachable"),
				))
			}

			return out
		}), 1)
}
