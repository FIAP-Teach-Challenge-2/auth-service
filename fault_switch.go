package main

import "sync/atomic"

type FaultSwitch struct {
	enabled atomic.Bool
}

func (f *FaultSwitch) Enable() {
	f.enabled.Store(true)
}

func (f *FaultSwitch) Disable() {
	f.enabled.Store(false)
}

func (f *FaultSwitch) Enabled() bool {
	return f.enabled.Load()
}
