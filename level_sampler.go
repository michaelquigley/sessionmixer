package sessionmixer

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/michaelquigley/scarlettctl"
	"github.com/michaelquigley/sessionmixer/topology"
)

// match the normal 30 Hz draw cadence without tying hardware reads to redraws.
const levelSampleInterval = time.Second / 30

type levelReader interface {
	GetValue() (int64, error)
}

type levelSample struct {
	index   int
	control levelReader
}

type levelSampler struct {
	state   *topology.DeviceState
	samples []levelSample
}

// StartMetering samples configured meters in the background using a separate ALSA
// handle. The returned stop function joins the worker before closing its handle;
// callers must invoke it before releasing the session.
func (sm *SessionMixer) StartMetering() (func(), error) {
	requested := sm.meterControls()
	if len(requested) == 0 {
		return func() {}, nil
	}

	// don't share the event monitor and fader writes' handle with the sampler.
	card, err := scarlettctl.OpenCard(sm.session.Card.Number)
	if err != nil {
		return nil, fmt.Errorf("opening metering connection: %w", err)
	}
	controls, err := card.GetControls()
	if err != nil {
		card.Close()
		return nil, fmt.Errorf("discovering metering controls: %w", err)
	}
	sampler, err := newLevelSampler(sm.session.State, requested, controls)
	if err != nil {
		card.Close()
		return nil, err
	}
	stop := sampler.start()
	return sync.OnceFunc(func() {
		stop()
		card.Close()
	}), nil
}

func (sm *SessionMixer) meterControls() map[int]*scarlettctl.Control {
	requested := make(map[int]*scarlettctl.Control)
	for _, cueMix := range sm.cueMixUIs {
		for i, index := range cueMix.masterUI.levelIndices {
			requested[index] = cueMix.masterUI.levelControls[i]
		}
		for _, fader := range cueMix.deviceFaders {
			for i, index := range fader.levelIndices {
				requested[index] = fader.levelControls[i]
			}
		}
	}
	return requested
}

func newLevelSampler(state *topology.DeviceState, requested map[int]*scarlettctl.Control, controls []*scarlettctl.Control) (*levelSampler, error) {
	byID := make(map[string]*scarlettctl.Control, len(controls))
	for _, control := range controls {
		byID[control.FullID()] = control
	}
	sampler := &levelSampler{state: state}
	indices := make([]int, 0, len(requested))
	for index := range requested {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	for _, index := range indices {
		id := requested[index].FullID()
		control := byID[id]
		if control == nil {
			return nil, fmt.Errorf("metering control %q unavailable", id)
		}
		sampler.samples = append(sampler.samples, levelSample{index: index, control: control})
	}
	return sampler, nil
}

func (s *levelSampler) start() func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(levelSampleInterval)
		defer ticker.Stop()
		for {
			s.sample(stop)
			select {
			case <-stop:
				return
			case <-ticker.C:
			}
		}
	}()
	return sync.OnceFunc(func() {
		close(stop)
		<-done
	})
}

func (s *levelSampler) sample(stop <-chan struct{}) {
	for _, sample := range s.samples {
		select {
		case <-stop:
			return
		default:
		}
		value, err := sample.control.GetValue()
		if err != nil {
			// an unreadable meter displays silence, rather than a stale peak.
			value = 0
		}
		s.state.UpdateLevelMeter(sample.index, value)
	}
}
