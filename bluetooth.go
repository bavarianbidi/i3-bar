package main

import (
	"github.com/barista-run/barista/bar"
	"github.com/barista-run/barista/colors"
	"github.com/barista-run/barista/modules/bluetooth"
	"github.com/barista-run/barista/modules/meta/split"
	"github.com/barista-run/barista/outputs"
	"github.com/barista-run/barista/pango"
)

func bluetoothAudio(adapter, address, icon string) (bar.Module, bar.Module) {
	return split.New(bluetooth.Device(adapter, address).Output(func(b bluetooth.DeviceInfo) bar.Output {

		out := outputs.Group()

		switch b.Connected {
		case true:
			color := colorOn
			iconAppendix := ""

			// to get the battery status
			// /etc/bluetooth/main.conf requires "Experimental = true"
			//
			// change icon color if battery is low
			if b.Battery <= 20 {
				color = colorBatteryLow
			}

			// summary
			out.Append(outputs.Pango(
				pango.Icon("mdi-" + icon + iconAppendix).Alpha(0.6).Color(colors.Hex(color)),
			))

			// detail
			out.Append(outputs.Pango(
				pango.Icon("mdi-"+icon).Alpha(0.6).Color(colors.Hex(color)),
				spacer,
				pango.Text(b.Name),
			))

			out.Append(outputs.Pango(
				pango.Icon("mdi-battery").Alpha(0.6),
				pango.Textf("%d%%", b.Battery),
			))

		case false:
			color := colorOff
			iconAppendix := "-off"

			// summary
			out.Append(outputs.Pango(
				pango.Icon("mdi-" + icon + iconAppendix).Alpha(0.6).Color(colors.Hex(color)),
			))

			// detail
			out.Append(outputs.Pango(
				pango.Icon("mdi-"+icon).Alpha(0.6).Color(colors.Hex(color)),
				spacer,
				pango.Text(b.Name),
			))

		}

		return out
	}), 1)
}
