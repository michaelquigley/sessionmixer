package sessionmixer

import (
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/michaelquigley/scarlettctl"
	"github.com/michaelquigley/sessionmixer/session"
	"github.com/michaelquigley/sessionmixer/topology"
	"github.com/michaelquigley/sessionmixer/topology/profiles"
)

func meterTestState() *topology.DeviceState {
	return topology.NewDeviceState(&topology.Device{
		Profile: profiles.NewScarlett16i16Gen4Profile(topology.FirmwareVersion{}, topology.FirmwareVersion{}),
	})
}

func meterTestControl(index int) *scarlettctl.Control {
	return &scarlettctl.Control{
		NumID: 1, Name: "Level Meter", Interface: scarlettctl.InterfacePCM,
		Index: index, Count: 54, Type: scarlettctl.ControlTypeInteger, Max: 32767,
	}
}

func TestMeterSelectionAndConnection(t *testing.T) {
	left, right, mono := meterTestControl(0), meterTestControl(1), meterTestControl(2)
	mixL, mixR := meterTestControl(34), meterTestControl(35)
	state := meterTestState()
	stereoDevice := &session.Device{Config: &session.DeviceConfig{Name: "stereo"}, Ports: []*topology.Port{
		{LevelMeterControl: left, LevelMeterIndex: 0},
		{LevelMeterControl: right, LevelMeterIndex: 1},
	}}
	monoDevice := &session.Device{Config: &session.DeviceConfig{Name: "mono"}, Ports: []*topology.Port{{LevelMeterControl: mono, LevelMeterIndex: 2}}}
	cue := &session.CueMix{
		Config: &session.CueMixConfig{Name: "cue"},
		MixL:   &topology.Mix{OutputLevelMeterControl: mixL, OutputLevelMeterIndex: 34},
		MixR:   &topology.Mix{OutputLevelMeterControl: mixR, OutputLevelMeterIndex: 35},
		Faders: []*session.DeviceFader{{Device: stereoDevice}, {Device: monoDevice}},
	}
	// the same inputs in two cue mixes must still be sampled only once.
	mixer := NewSessionMixer(&session.Session{
		Config: &session.SessionConfig{}, State: state, CueMixes: []*session.CueMix{cue, cue},
	}, nil)
	requested := mixer.meterControls()
	if len(requested) != 5 {
		t.Fatalf("requested %d meters, want 5", len(requested))
	}
	// controls discovered on the metering connection have the same full IDs but
	// different pointers; every read must use the new connection's control.
	available := []*scarlettctl.Control{
		meterTestControl(0), meterTestControl(1), meterTestControl(2),
		meterTestControl(34), meterTestControl(35), meterTestControl(53),
	}
	sampler, err := newLevelSampler(state, requested, available)
	if err != nil {
		t.Fatal(err)
	}
	for i, sample := range sampler.samples {
		if sample.control != available[i] {
			t.Errorf("sample %d did not use its metering connection control", i)
		}
	}
	if _, err := newLevelSampler(state, requested, available[:4]); err == nil {
		t.Fatal("missing configured meter did not fail startup")
	}
}

type testLevelReader func() (int64, error)

func (read testLevelReader) GetValue() (int64, error) { return read() }

func TestLevelSamplerPublishesAndClearsErrors(t *testing.T) {
	state := meterTestState()
	state.UpdateLevelMeter(1, 32767)
	sampler := &levelSampler{state: state, samples: []levelSample{
		{index: 0, control: testLevelReader(func() (int64, error) { return 1234, nil })},
		{index: 1, control: testLevelReader(func() (int64, error) { return 99, errors.New("read failed") })},
	}}
	sampler.sample(make(chan struct{}))
	if got := state.GetLevelMeter(0); got != 1234 {
		t.Fatalf("sampled level = %d, want 1234", got)
	}
	if got := state.GetLevelMeter(1); got != 0 {
		t.Fatalf("failed read left level = %d, want silence", got)
	}
}

func TestMeterCacheRemainsReadableDuringBlockedHardwareRead(t *testing.T) {
	state := meterTestState()
	state.UpdateLevelMeter(0, 32767)
	entered, release := make(chan struct{}), make(chan struct{})
	var enteredOnce, releaseOnce sync.Once
	releaseRead := func() { releaseOnce.Do(func() { close(release) }) }
	sampler := &levelSampler{state: state, samples: []levelSample{{
		index: 0, control: testLevelReader(func() (int64, error) {
			enteredOnce.Do(func() { close(entered) })
			<-release
			return 16384, nil
		}),
	}}}
	stop := sampler.start()
	defer stop()
	defer releaseRead()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("sampler did not start")
	}
	// these disconnected controls would return errors if the UI read hardware.
	control := meterTestControl(0)
	fader := &DeviceFaderUI{state: state, levelControls: []*scarlettctl.Control{control}, levelIndices: []int{0}, levelMax: control.Max}
	master := &MasterUI{state: state, levelControls: []*scarlettctl.Control{control}, levelIndices: []int{0}, levelMax: control.Max}
	for range 100 {
		if got := fader.getNormalizedLevels()[0]; got != 1 {
			t.Fatalf("cached fader level = %v, want 1", got)
		}
		if got := master.getNormalizedLevels()[0]; got != 1 {
			t.Fatalf("cached master level = %v, want 1", got)
		}
	}
	stopped := make(chan struct{})
	go func() { stop(); close(stopped) }()
	select {
	case <-stopped:
		t.Fatal("stop returned while hardware read was still running")
	case <-time.After(10 * time.Millisecond):
	}
	releaseRead()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("stop did not join the worker")
	}
	if got := state.GetLevelMeter(0); got != 16384 {
		t.Fatalf("last level = %d, want 16384", got)
	}
	want := float32((20*math.Log10(16384.0/32767) + 96) / 96)
	if got := master.getNormalizedLevels()[0]; math.Abs(float64(got-want)) > 1e-6 {
		t.Fatalf("cached normalized level = %v, want %v", got, want)
	}
	stop() // repeated shutdown is harmless.
}
