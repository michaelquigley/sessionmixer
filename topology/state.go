package topology

import (
	"sync"
	"sync/atomic"
)

// DeviceState tracks the dynamic runtime state of a Device.
// This is separate from the Device structure which represents static topology.
type DeviceState struct {
	device *Device

	// Level meter values (updated frequently)
	levelMeterValues []int64

	// Gang state for mixes (mix ID -> partner mix ID)
	gangState sync.Map
}

// NewDeviceState creates a new state tracker for a device
func NewDeviceState(device *Device) *DeviceState {
	return &DeviceState{
		device:           device,
		levelMeterValues: make([]int64, device.Profile.LevelMeterCount()),
	}
}

// Device returns the underlying device
func (s *DeviceState) Device() *Device {
	return s.device
}

// UpdateLevelMeter atomically updates a level meter value
func (s *DeviceState) UpdateLevelMeter(index int, value int64) {
	if index >= 0 && index < len(s.levelMeterValues) {
		atomic.StoreInt64(&s.levelMeterValues[index], value)
	}
}

// GetLevelMeter atomically reads a level meter value
func (s *DeviceState) GetLevelMeter(index int) int64 {
	if index >= 0 && index < len(s.levelMeterValues) {
		return atomic.LoadInt64(&s.levelMeterValues[index])
	}
	return 0
}

// GetPortLevel returns the level for a port (by ID)
func (s *DeviceState) GetPortLevel(portID string) int64 {
	port := s.device.GetPort(portID)
	if port == nil || port.LevelMeterIndex < 0 {
		return 0
	}
	return s.GetLevelMeter(port.LevelMeterIndex)
}

// GetMixOutputLevel returns the output level for a mix (by ID)
func (s *DeviceState) GetMixOutputLevel(mixID string) int64 {
	mix := s.device.GetMix(mixID)
	if mix == nil || mix.OutputLevelMeterIndex < 0 {
		return 0
	}
	return s.GetLevelMeter(mix.OutputLevelMeterIndex)
}

// ReadAllLevels bulk-reads all level meters from hardware
// This is more efficient than individual reads for UI updates
func (s *DeviceState) ReadAllLevels() error {
	for i, ctl := range s.device.LevelMeterControls {
		if ctl != nil {
			val, err := ctl.GetValue()
			if err != nil {
				continue
			}
			s.UpdateLevelMeter(i, val)
		}
	}
	return nil
}

// RefreshAllRouting refreshes all routing endpoint states from hardware
func (s *DeviceState) RefreshAllRouting() error {
	for _, ep := range s.device.AllRoutingEndpoints() {
		if err := ep.RefreshFromHardware(); err != nil {
			// Continue on error - don't fail the whole refresh
			continue
		}
	}
	return nil
}

// GangMixes sets two mixes as a stereo pair
func (s *DeviceState) GangMixes(mixA, mixB string) {
	s.gangState.Store(mixA, mixB)
	s.gangState.Store(mixB, mixA)
}

// UngangMix removes a mix from its stereo pair
func (s *DeviceState) UngangMix(mixID string) {
	if partner, ok := s.gangState.Load(mixID); ok {
		s.gangState.Delete(mixID)
		s.gangState.Delete(partner.(string))
	}
}

// IsGanged returns true if the mix is part of a stereo pair
func (s *DeviceState) IsGanged(mixID string) bool {
	_, ok := s.gangState.Load(mixID)
	return ok
}

// GetGangPartner returns the partner mix ID if ganged, empty string otherwise
func (s *DeviceState) GetGangPartner(mixID string) string {
	if partner, ok := s.gangState.Load(mixID); ok {
		return partner.(string)
	}
	return ""
}

// GetAllGangedPairs returns all ganged mix pairs as a slice of [2]string
// Each pair appears only once (not duplicated in reverse order)
func (s *DeviceState) GetAllGangedPairs() [][2]string {
	seen := make(map[string]bool)
	var pairs [][2]string

	s.gangState.Range(func(key, value interface{}) bool {
		mixA := key.(string)
		mixB := value.(string)

		// Only add if we haven't seen either mix in a pair yet
		if !seen[mixA] && !seen[mixB] {
			pairs = append(pairs, [2]string{mixA, mixB})
			seen[mixA] = true
			seen[mixB] = true
		}
		return true
	})

	return pairs
}

// LevelMeterSnapshot returns a copy of all level meter values
// This is useful for consistent reads across multiple meters
func (s *DeviceState) LevelMeterSnapshot() []int64 {
	snapshot := make([]int64, len(s.levelMeterValues))
	for i := range s.levelMeterValues {
		snapshot[i] = atomic.LoadInt64(&s.levelMeterValues[i])
	}
	return snapshot
}
