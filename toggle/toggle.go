package toggle

import (
	"time"

	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/timing"
)

// Module presents a toggle-able module
type Module struct {
	toggleValue value.Value // of string
	toggleFunc  func() string
	clickFunc   func()
	outputFunc  func(string) *bar.Segment
	scheduler   *timing.Scheduler
}

func New(toggleFunc func() string, clickFunc func(), outputFunc func(string) *bar.Segment, every time.Duration) *Module {
	return &Module{
		toggleFunc: toggleFunc,
		clickFunc:  clickFunc,
		outputFunc: outputFunc,
		scheduler:  timing.NewScheduler().Every(every),
	}
}

func (m *Module) Stream(s bar.Sink) {
	m.refresh()
	toggleValue := m.toggleValue.Get().(string)
	toggleValueSub, done := m.toggleValue.Subscribe()
	defer done()
	for {
		s.Output(
			m.outputFunc(toggleValue).OnClick(m.click),
		)
		select {
		case <-toggleValueSub:
			toggleValue = m.toggleValue.Get().(string)
		case <-m.scheduler.C:
			m.refresh()
		}
	}
}

func (m *Module) click(e bar.Event) {
	m.clickFunc()
	m.refresh()
}

func (m *Module) refresh() {
	v := m.toggleFunc()
	m.toggleValue.Set(v)
}
