package main

import (
	"time"

	"github.com/barista-run/barista/bar"
	"github.com/barista-run/barista/base/click"
	"github.com/barista-run/barista/format"
	"github.com/barista-run/barista/modules/cputemp"
	"github.com/barista-run/barista/modules/meminfo"
	"github.com/barista-run/barista/modules/sysinfo"
	"github.com/barista-run/barista/outputs"
	"github.com/barista-run/barista/pango"
	"github.com/martinlindhe/unit"
)

func freeMem() *meminfo.Module {
	return meminfo.New().Output(func(m meminfo.Info) bar.Output {
		out := outputs.Pango(
			pango.Icon("material-memory").Alpha(0.8),
			format.IBytesize(m.Available()),
		)
		freeGigs := m.Available().Gigabytes()
		threshold(out,
			freeGigs < 0.5,
			freeGigs < 1,
			freeGigs < 2,
			freeGigs > 12)
		out.OnClick(click.Left(func() {
			mainModalController.Toggle("sysinfo")
		}))
		return out
	})
}

func swapInfo() *meminfo.Module {
	return meminfo.New().Output(func(m meminfo.Info) bar.Output {
		return outputs.Pango(
			pango.Icon("mdi-swap-horizontal").Alpha(0.8),
			format.IBytesize(m["SwapTotal"]-m["SwapFree"]), spacer,
			pango.Textf("(% 2.0f%%)", (1-m.FreeFrac("Swap"))*100.0).Small(),
		)
	})
}

func uptime() *sysinfo.Module {
	return sysinfo.New().Output(func(s sysinfo.Info) bar.Output {
		u := s.Uptime
		var uptimeOut *pango.Node
		if u.Hours() < 24 {
			uptimeOut = pango.Textf("%d:%02d",
				int(u.Hours()), int(u.Minutes())%60)
		} else {
			uptimeOut = pango.Textf("%dd%02dh",
				int(u.Hours()/24), int(u.Hours())%24)
		}
		return pango.Icon("mdi-trending-up").Alpha(0.6).Concat(uptimeOut)
	})
}

func loadAvgDetail() *sysinfo.Module {
	return sysinfo.New().Output(func(s sysinfo.Info) bar.Output {
		return pango.Textf("%0.2f %0.2f", s.Loads[1], s.Loads[2]).Smaller()
	})
}

func loadAvg() *sysinfo.Module {
	return sysinfo.New().Output(func(s sysinfo.Info) bar.Output {
		out := outputs.Pango(
			pango.Icon("mdi-desktop-tower").Alpha(0.6),
			pango.Textf("%0.2f", s.Loads[0]),
		)
		// Load averages are unusually high for a few minutes after boot.
		if s.Uptime < 10*time.Minute {
			// so don't add colours until 10 minutes after system start.
			return out
		}
		threshold(out,
			s.Loads[0] > 128 || s.Loads[2] > 64,
			s.Loads[0] > 64 || s.Loads[2] > 32,
			s.Loads[0] > 32 || s.Loads[2] > 16,
		)
		out.OnClick(click.Left(func() {
			mainModalController.Toggle("sysinfo")
		}))
		return out
	})
}

func cpuTemp() *cputemp.Module {
	return cputemp.New().
		RefreshInterval(2 * time.Second).
		Output(func(temp unit.Temperature) bar.Output {
			out := outputs.Pango(
				pango.Icon("mdi-fan").Alpha(0.6), spacer,
				pango.Textf("%2d℃", int(temp.Celsius())),
			)
			threshold(out,
				temp.Celsius() > 90,
				temp.Celsius() > 70,
				temp.Celsius() > 60,
			)
			return out
		})
}
