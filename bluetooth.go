package main

import (
	"barista.run/bar"
	"barista.run/colors"
	"barista.run/modules/bluetooth"
	"barista.run/modules/meta/split"
	"barista.run/outputs"
	"barista.run/pango"
)

func bluetoothAudio(adapter, address, icon string) (bar.Module, bar.Module) {
	return split.New(bluetooth.Device(adapter, address).Output(func(b bluetooth.DeviceInfo) bar.Output {

		out := outputs.Group()

		color := colorOn
		iconAppendix := ""

		if !b.Connected {
			color = colorOff
			iconAppendix = "-off"
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

		if b.Connected {
			out.Append(outputs.Pango(
				pango.Icon("mdi-battery").Alpha(0.6),
				pango.Textf("%s: %d%%", b.Alias, b.Battery),
			))
		}

		return out
	}), 1)
}
