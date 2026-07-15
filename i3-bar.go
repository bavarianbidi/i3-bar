// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// sample-bar demonstrates a sample i3bar built using barista.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/barista-run/barista"
	"github.com/barista-run/barista/bar"
	"github.com/barista-run/barista/base/click"
	"github.com/barista-run/barista/base/watchers/netlink"
	"github.com/barista-run/barista/colors"
	"github.com/barista-run/barista/format"
	"github.com/barista-run/barista/group/modal"
	"github.com/barista-run/barista/modules/battery"
	"github.com/barista-run/barista/modules/clock"
	"github.com/barista-run/barista/modules/diskio"
	"github.com/barista-run/barista/modules/diskspace"
	"github.com/barista-run/barista/modules/media"
	"github.com/barista-run/barista/modules/meta/split"
	"github.com/barista-run/barista/modules/netinfo"
	"github.com/barista-run/barista/modules/netspeed"
	"github.com/barista-run/barista/modules/shell"

	"github.com/barista-run/barista/modules/volume"
	"github.com/barista-run/barista/modules/volume/alsa" // libasound2-dev or libsdl2-dev
	"github.com/barista-run/barista/modules/wlan"
	"github.com/barista-run/barista/outputs"
	"github.com/barista-run/barista/pango"
	"github.com/barista-run/barista/pango/icons/mdi"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	forge "github.com/git-pkgs/forge"

	colorful "github.com/lucasb-eyer/go-colorful"
)

type config struct {
	Jira struct {
		Token   string `koanf:"token"`
		OpenURL string `koanf:"openURL"`
		Icon    string `koanf:"icon"`
	} `koanf:"jira"`
	Forges []struct {
		Host    string `koanf:"host"`
		OpenURL string `koanf:"openURL"`
		Icon    string `koanf:"icon"`
	} `koanf:"forges"`
}

var spacer = pango.Text(" ").XXSmall()
var mainModalController modal.Controller

func truncate(in string, l int) string {
	fromStart := false
	if l < 0 {
		fromStart = true
		l = -l
	}
	inLen := len([]rune(in))
	if inLen <= l {
		return in
	}
	if fromStart {
		return "⋯" + string([]rune(in)[inLen-l+1:])
	}
	return string([]rune(in)[:l-1]) + "⋯"
}

func hms(d time.Duration) (h int, m int, s int) {
	h = int(d.Hours())
	m = int(d.Minutes()) % 60
	s = int(d.Seconds()) % 60
	return
}

func formatMediaTime(d time.Duration) string {
	h, m, s := hms(d)
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func makeMediaIconAndPosition(m media.Info) *pango.Node {
	iconAndPosition := pango.Icon("mdi-music-circle").Color(colors.Hex("#f70"))
	if m.PlaybackStatus == media.Playing {
		iconAndPosition.Append(spacer,
			pango.Textf("%s/", formatMediaTime(m.Position())))
	}
	if m.PlaybackStatus == media.Paused || m.PlaybackStatus == media.Playing {
		iconAndPosition.Append(spacer,
			pango.Textf("%s", formatMediaTime(m.Length)))
	}
	return iconAndPosition
}

func mediaFormatFunc(m media.Info) bar.Output {
	if m.PlaybackStatus == media.Stopped || m.PlaybackStatus == media.Disconnected {
		return nil
	}
	artist := truncate(m.Artist, 35)
	title := truncate(m.Title, 70-len(artist))
	if len(title) < 35 {
		artist = truncate(m.Artist, 35-len(title))
	}
	var iconAndPosition bar.Output
	if m.PlaybackStatus == media.Playing {
		iconAndPosition = outputs.Repeat(func(time.Time) bar.Output {
			return makeMediaIconAndPosition(m)
		}).Every(time.Second)
	} else {
		iconAndPosition = makeMediaIconAndPosition(m)
	}
	return outputs.Group(iconAndPosition, outputs.Pango(title, " - ", artist))
}

func home(path ...string) string {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	args := append([]string{usr.HomeDir}, path...)
	return filepath.Join(args...)
}

func deviceForMountPath(path string) string {
	mnt, _ := exec.Command("df", "-P", path).Output()
	lines := strings.Split(string(mnt), "\n")
	if len(lines) > 1 {
		devAlias := strings.Split(lines[1], " ")[0]
		dev, _ := exec.Command("realpath", devAlias).Output()
		devStr := strings.TrimSpace(string(dev))
		if devStr != "" {
			return devStr
		}
		return devAlias
	}
	return ""
}

func makeIconOutput(key string) *bar.Segment {
	return outputs.Pango(spacer, pango.Icon(key), spacer)
}

func threshold(out *bar.Segment, urgent bool, color ...bool) *bar.Segment {
	if urgent {
		return out.Urgent(true)
	}
	colorKeys := []string{"bad", "degraded", "good"}
	for i, c := range colorKeys {
		if len(color) > i && color[i] {
			return out.Color(colors.Scheme(c))
		}
	}
	return out
}

func main() {
	err := mdi.Load(home("go/src/github.com/Templarian/MaterialDesign-Webfont"))
	if err != nil {
		log.Fatal(err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// read config file
	k := koanf.New(".")
	if err := k.Load(file.Provider(homeDir+"/.config/i3/config.yaml"), yaml.Parser()); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	cfg := config{}
	if err := k.Unmarshal("", &cfg); err != nil {
		log.Fatalf("error unmarshaling config: %v", err)
	}

	colors.LoadBarConfig()
	bg := colors.Scheme("background")
	fg := colors.Scheme("statusline")
	if fg != nil && bg != nil {
		_, _, v := fg.Colorful().Hsv()
		if v < 0.3 {
			v = 0.3
		}
		colors.Set("bad", colorful.Hcl(40, 1.0, v).Clamped())
		colors.Set("degraded", colorful.Hcl(90, 1.0, v).Clamped())
		colors.Set("good", colorful.Hcl(120, 1.0, v).Clamped())
	}

	localdate := clock.Local().
		Output(time.Second, func(now time.Time) bar.Output {
			return outputs.Pango(
				pango.Icon("material-today").Alpha(0.6),
				now.Format("Mon Jan 2"),
			)
		})

	localtime := clock.Local().
		Output(time.Second, func(now time.Time) bar.Output {
			return outputs.Text(now.Format("15:04")).
				OnClick(click.Left(func() {
					mainModalController.Toggle("timezones")
				}))
		})

	makeTzClock := func(lbl, tzName string) bar.Module {
		c, err := clock.ZoneByName(tzName)
		if err != nil {
			panic(err)
		}
		return c.Output(time.Minute, func(now time.Time) bar.Output {
			return outputs.Pango(pango.Text(lbl).Smaller(), spacer, now.Format("15:04"))
		})
	}

	var forges []bar.Module
	for _, frg := range cfg.Forges {
		notification := shell.New(homeDir+"/go/bin/forge", "notification", "list", "--host", frg.Host, "--unread", "-o", "json").
			Output(func(s string) bar.Output {
				if s == "" {
					return nil
				}

				var parsed []forge.Notification
				if err := json.Unmarshal([]byte(s), &parsed); err != nil {
					return outputs.Text(s)
				}

				color := colors.Hex(colorOn)
				if len(parsed) > 0 {
					color = colors.Hex(colorOff)
				}

				// return outputs.Text(fmt.Sprintf("%d", len(parsed)))
				return outputs.Pango(
					pango.Icon(frg.Icon).Alpha(0.6),
					spacer,
					pango.Textf("%d", len(parsed)),
				).Color(color).
					OnClick(click.Left(func() {
						_ = exec.Command("xdg-open", frg.OpenURL).Start()
					}))
			}).Every(time.Duration(5) * time.Minute)

		forges = append(forges, notification)
	}

	openJiraAlerts := shell.New("/home/comario/bin/jira", "alert", "list", "--since", "5d", "--json").
		Output(func(s string) bar.Output {
			if s == "" {
				return nil
			}

			var parsed []map[string]interface{}
			if err := json.Unmarshal([]byte(s), &parsed); err != nil {
				return outputs.Text(s)
			}

			color := colors.Hex(colorOn)
			if len(parsed) > 0 {
				color = colors.Hex(colorOff)
			}

			return outputs.Pango(
				pango.Icon(cfg.Jira.Icon).Alpha(0.6),
				spacer,
				pango.Textf("%d", len(parsed)),
			).Color(color).
				OnClick(click.Left(func() {
					_ = exec.Command("xdg-open", cfg.Jira.OpenURL).Start()
				}))
		}).Every(time.Duration(10) * time.Minute).WithEnv(fmt.Sprint("JIRA_API_TOKEN=" + cfg.Jira.Token))

	zscallerStatus := shell.New("sh", "-c", "ip route show dev zcctun0 || echo __ZSCALER_ERROR__").
		Output(func(s string) bar.Output {
			if s == "" {
				return nil
			}

			if strings.Contains(s, "__ZSCALER_ERROR__") {
				return outputs.Pango(
					pango.Icon("mdi-tunnel-outline").Alpha(0.6),
				).Color(colors.Hex(colorOff))
			}

			length := len(strings.Split(s, "\n"))

			color := colors.Hex(colorOn)
			if length <= 3 {
				color = colors.Hex(colorOff)
			}

			return outputs.Pango(
				pango.Icon("mdi-tunnel-outline").Alpha(0.6),
			).Color(color)
		}).Every(time.Duration(2) * time.Minute)

	battSummary, battDetail := split.New(battery.All().Output(func(i battery.Info) bar.Output {
		if i.Status == battery.Disconnected || i.Status == battery.Unknown {
			return nil
		}
		iconName := "battery"
		if i.Status == battery.Charging {
			iconName += "-charging"
		}
		tenth := i.RemainingPct() / 10
		switch {
		case tenth == 0:
			iconName += "-outline"
		case tenth < 10:
			iconName += fmt.Sprintf("-%d0", tenth)
		}
		mainModalController.SetOutput("battery", makeIconOutput("mdi-"+iconName))
		rem := i.RemainingTime()
		out := outputs.Group()
		// First segment will be used in summary mode.
		out.Append(outputs.Pango(
			pango.Icon("mdi-"+iconName).Alpha(0.6),
			pango.Textf("%d:%02d", int(rem.Hours()), int(rem.Minutes())%60),
		).OnClick(click.Left(func() {
			mainModalController.Toggle("battery")
		})))
		// Others in detail mode.
		out.Append(outputs.Pango(
			pango.Icon("mdi-"+iconName).Alpha(0.6),
			pango.Textf("%d%%", i.RemainingPct()),
			spacer,
			pango.Textf("(%d:%02d)", int(rem.Hours()), int(rem.Minutes())%60),
		).OnClick(click.Left(func() {
			mainModalController.Toggle("battery")
		})))
		out.Append(outputs.Pango(
			pango.Textf("%4.1f/%4.1f", i.EnergyNow, i.EnergyFull),
			pango.Text("Wh").Smaller(),
		))
		out.Append(outputs.Pango(
			pango.Textf("% +6.2f", i.SignedPower()),
			pango.Text("W").Smaller(),
		))
		switch {
		case i.RemainingPct() <= 5:
			out.Urgent(true)
		case i.RemainingPct() <= 15:
			out.Color(colors.Scheme("bad"))
		case i.RemainingPct() <= 25:
			out.Color(colors.Scheme("degraded"))
		}
		return out
	}), 1)

	wifiName, wifiDetails := split.New(wlan.Any().Output(func(i wlan.Info) bar.Output {
		if !i.Connecting() && !i.Connected() {
			mainModalController.SetOutput("network", makeIconOutput("mdi-ethernet"))
			return nil
		}
		mainModalController.SetOutput("network", makeIconOutput("mdi-wifi"))
		if i.Connecting() {
			return outputs.Pango(pango.Icon("mdi-wifi").Alpha(0.6), "...").
				Color(colors.Scheme("degraded"))
		}
		out := outputs.Group()
		// First segment shown in summary mode only.
		out.Append(outputs.Pango(
			pango.Icon("mdi-wifi").Alpha(0.6),
			pango.Text(truncate(i.SSID, -9)),
		).OnClick(click.Left(func() {
			mainModalController.Toggle("network")
		})))
		// Full name, frequency, bssid in detail mode
		out.Append(outputs.Pango(
			pango.Icon("mdi-wifi").Alpha(0.6),
			pango.Text(i.SSID),
		))
		out.Append(outputs.Textf("%2.1fG", i.Frequency.Gigahertz()))
		out.Append(outputs.Pango(
			pango.Icon("mdi-access-point").Alpha(0.8),
			pango.Text(i.AccessPointMAC).Small(),
		))
		return out
	}), 1)

	vol := volume.New(alsa.DefaultMixer()).Output(func(v volume.Volume) bar.Output {
		if v.Mute {
			return outputs.
				Pango(pango.Icon("mdi-volume-mute").Alpha(0.8), spacer, "MUT").
				Color(colors.Scheme("degraded"))
		}
		iconName := "off"
		pct := v.Pct()
		if pct > 66 {
			iconName = "high"
		} else if pct > 33 {
			iconName = "low"
		}
		return outputs.Pango(
			pango.Icon("mdi-volume-"+iconName).Alpha(0.6),
			spacer,
			pango.Textf("%2d%%", pct),
		)
	})

	// System information modules
	loadAvg := loadAvg()
	loadAvgDetail := loadAvgDetail()
	uptime := uptime()
	freeMem := freeMem()
	swapMem := swapInfo()
	temp := cpuTemp()

	sub := netlink.Any()
	iface := sub.Get().Name
	sub.Unsubscribe()
	netsp := netspeed.New(iface).
		RefreshInterval(2 * time.Second).
		Output(func(s netspeed.Speeds) bar.Output {
			return outputs.Pango(
				pango.Icon("mdi-upload-network").Alpha(0.5), spacer, pango.Textf("%7s", format.Byterate(s.Tx)),
				pango.Text(" ").Small(),
				pango.Icon("mdi-download-network").Alpha(0.5), spacer, pango.Textf("%7s", format.Byterate(s.Rx)),
			)
		})

	net := netinfo.New().Output(func(i netinfo.State) bar.Output {
		if !i.Enabled() {
			return nil
		}
		if i.Connecting() || len(i.IPs) < 1 {
			return outputs.Text(i.Name).Color(colors.Scheme("degraded"))
		}
		return outputs.Group(outputs.Text(i.Name), outputs.Textf("%s", i.IPs[0]))
	})

	formatDiskSpace := func(i diskspace.Info, icon string) bar.Output {
		out := outputs.Pango(
			pango.Icon(icon).Alpha(0.7), spacer, format.IBytesize(i.Available))
		return threshold(out,
			i.Available.Gigabytes() < 1,
			i.AvailFrac() < 0.05,
			i.AvailFrac() < 0.1,
		)
	}

	rootDev := deviceForMountPath("/")
	var homeDiskspace bar.Module
	if deviceForMountPath(home()) != rootDev {
		homeDiskspace = diskspace.New(home()).Output(func(i diskspace.Info) bar.Output {
			return formatDiskSpace(i, "typecn-home-outline")
		})
	}
	rootDiskspace := diskspace.New("/").Output(func(i diskspace.Info) bar.Output {
		return formatDiskSpace(i, "mdi-harddisk")
	})

	mainDiskio := diskio.New(strings.TrimPrefix(rootDev, "/dev/")).
		Output(func(r diskio.IO) bar.Output {
			return pango.Icon("mdi-swap-vertical").
				Concat(spacer).
				ConcatText(format.IByterate(r.Total()))
		})

	mediaSummary, mediaDetail := split.New(media.New("spotify").Output(mediaFormatFunc), 1)

	mainModal := modal.New()

	sysMode := mainModal.Mode("sysinfo").
		SetOutput(makeIconOutput("mdi-poll")).
		Add(loadAvg).
		Detail(loadAvgDetail, uptime).
		Add(freeMem).
		Detail(swapMem, temp)
	if homeDiskspace != nil {
		sysMode.Detail(homeDiskspace)
	}
	sysMode.Detail(rootDiskspace, mainDiskio)

	mainModal.Mode("gitlab notifications").
		SetOutput(makeIconOutput("mdi-alert")).
		Add(forges...).Add(openJiraAlerts)

	// TODO:
	// bavarianbidi: read bluetooth devices from config file instead of hardcoding them here
	//
	// e.g.:
	// bluetooth:
	// - devide: id
	//   icon: headphones
	// - devide: id
	//   icon: speaker

	// headphones
	headsetSummary, headsetDetail := bluetoothAudio("hci0", "14:3F:A6:1B:FA:77", "headphones")

	// bluetooth box
	soundcoreSummary, soundcoreDetail := bluetoothAudio("hci0", "08:EB:ED:83:82:01", "speaker")

	mainModal.Mode("bluetooth-audio").
		SetOutput(makeIconOutput("mdi-bluetooth")).
		Add(soundcoreSummary).
		Add(headsetSummary).
		Detail(soundcoreDetail).
		Detail(headsetDetail)

	quickMillSummary, quickMillDetail := shellyStatus("192.168.178.64", "coffee")

	mainModal.Mode("shelly").
		SetOutput(makeIconOutput("mdi-coffee")).
		Add(quickMillSummary).
		Detail(quickMillDetail)

	mainModal.Mode("VPN").
		SetOutput(makeIconOutput("mdi-tunnel-outline")).
		// Summary().
		// Detail().
		Add(zscallerStatus)

	mainModal.Mode("network").
		SetOutput(makeIconOutput("mdi-ethernet")).
		Summary(wifiName).
		Detail(wifiDetails, net, netsp)

	mainModal.Mode("media").
		SetOutput(makeIconOutput("mdi-music-box")).
		Add(vol, mediaSummary, mediaDetail)

	mainModal.Mode("battery").
		// Filled in by the battery module if one is available.
		SetOutput(nil).
		Summary(battSummary).
		Detail(battDetail)

	mainModal.Mode("timezones").
		SetOutput(makeIconOutput("mdi-earth")).
		Detail(makeTzClock("Seattle", "America/Los_Angeles")).
		Detail(makeTzClock("New York", "America/New_York")).
		Detail(makeTzClock("UTC", "Etc/UTC")).
		Detail(makeTzClock("Berlin", "Europe/Berlin")).
		Detail(makeTzClock("Bangalore", "Asia/Kolkata"))

	var mm bar.Module
	mm, mainModalController = mainModal.Build()
	barista.SuppressSignals(true)
	panic(barista.Run(mm, localdate, localtime))
}
