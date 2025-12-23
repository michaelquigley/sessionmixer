package sessionmixer

import (
	"fmt"
	"math"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/michaelquigley/dfx"
	"github.com/michaelquigley/scarlettctl"
	"github.com/michaelquigley/sessionmixer/session"
	"github.com/michaelquigley/sessionmixer/topology"
)

// DeviceFaderUI renders a device fader with VU meters.
type DeviceFaderUI struct {
	// The underlying device fader
	fader *session.DeviceFader

	// Device state for level meter access
	state *topology.DeviceState

	// Fader parameters
	params dfx.FaderParams

	// VU meter for device input levels
	vuMeter *dfx.VUMeter

	// VU waterfall for level history (optional, toggled by click)
	vuWaterfall *dfx.VUWaterfall

	// Whether waterfall is currently shown
	showWaterfall bool

	// Level meter controls (cached from device ports)
	levelControls []*scarlettctl.Control

	// Level smoothing buffers
	levelBuffers []*RingBuffer[int64]

	// Level min/max from first control
	levelMin, levelMax int64
}

// NewDeviceFaderUI creates a new UI component for a device fader.
func NewDeviceFaderUI(fader *session.DeviceFader, state *topology.DeviceState, levelSmoothing int, showWaterfall bool) *DeviceFaderUI {
	ui := &DeviceFaderUI{
		fader:         fader,
		state:         state,
		showWaterfall: showWaterfall,
	}

	// Collect level meter controls from device ports
	device := fader.Device
	for _, port := range device.Ports {
		if port.LevelMeterControl != nil {
			ui.levelControls = append(ui.levelControls, port.LevelMeterControl)
		}
	}

	// Get level range from first control
	if len(ui.levelControls) > 0 {
		ui.levelMin = ui.levelControls[0].Min
		ui.levelMax = ui.levelControls[0].Max
	}

	// Create level smoothing buffers if enabled
	if levelSmoothing > 1 && len(ui.levelControls) > 0 {
		ui.levelBuffers = make([]*RingBuffer[int64], len(ui.levelControls))
		for i := range ui.levelBuffers {
			ui.levelBuffers[i] = NewRingBuffer[int64](levelSmoothing)
		}
	}

	// Create VU meter if we have level controls
	if len(ui.levelControls) > 0 {
		ui.vuMeter = dfx.NewVUMeter(len(ui.levelControls))
		ui.vuMeter.Height = 300
		ui.vuMeter.Mode = dfx.VUMeterHighres

		// Label channels
		labels := make([]string, len(ui.levelControls))
		if len(labels) == 1 {
			labels[0] = "M" // Mono
		} else if len(labels) == 2 {
			labels[0] = "L"
			labels[1] = "R"
		} else {
			for i := range labels {
				labels[i] = fmt.Sprintf("%d", i+1)
			}
		}
		ui.vuMeter.SetLabels(labels)

		// Create VU waterfall (same channel count, matching height)
		ui.vuWaterfall = dfx.NewVUWaterfall(len(ui.levelControls))
		ui.vuWaterfall.Height = 300
		ui.vuWaterfall.ChannelWidth = 12
		ui.vuWaterfall.ChannelGap = 4
		ui.vuWaterfall.RowHeight = 1
		ui.vuWaterfall.Highres = true
		ui.vuWaterfall.SetHistorySize(300)
	}

	// Configure fader parameters with dB taper and display
	ui.params = ui.createFaderParams()

	return ui
}

// createFaderParams creates dfx.FaderParams for the device fader
func (ui *DeviceFaderUI) createFaderParams() dfx.FaderParams {
	// Use 72dB taper for mixer volumes
	taper := dfx.DecibelTaper(72)

	min := float32(ui.fader.Min())
	max := float32(ui.fader.Max())

	params := dfx.FaderParams{
		Width:       33.0,
		Height:      300.0,
		ShowTooltip: true,
		Taper:       taper,
		Format: func(normalized float32) string {
			rawValue := normalized*(max-min) + min

			// Handle mute/zero case
			if rawValue <= min {
				return "-∞ dB"
			}

			// Logarithmic conversion: 0 to max maps to -∞ to +12 dB
			db := 20.0*math.Log10(float64(rawValue)/float64(max)) + 12.0
			return fmt.Sprintf("%.1f dB", db)
		},
	}

	return params
}

// Draw renders the device fader.
// Returns the column width used.
func (ui *DeviceFaderUI) Draw(state *dfx.State) float32 {
	name := ui.fader.Name()
	currentValue := ui.fader.GetCurrentValue()
	min := ui.fader.Min()
	max := ui.fader.Max()

	// Update VU meter levels
	levels := ui.getNormalizedLevels()
	if ui.vuMeter != nil {
		ui.vuMeter.SetLevels(levels)
	}
	if ui.vuWaterfall != nil {
		ui.vuWaterfall.SetLevels(levels)
	}

	// Calculate column width
	faderWidth := float32(60.0)
	vuWidth := float32(0)
	if ui.vuMeter != nil {
		vuWidth = ui.vuMeter.Width() + 8 // 8px gap
	}
	waterfallWidth := float32(0)
	if ui.showWaterfall && ui.vuWaterfall != nil {
		waterfallWidth = ui.vuWaterfall.Width() + 8 // 8px gap
	}
	columnWidth := faderWidth + vuWidth + waterfallWidth + 16 // 16px padding

	// Draw label (click to toggle waterfall)
	imgui.Text(name)
	if imgui.IsItemClicked() {
		ui.showWaterfall = !ui.showWaterfall
	}

	params := ui.params
	newValue, changed := dfx.FaderI(
		fmt.Sprintf("##fader_%p", ui),
		int(currentValue),
		int(min),
		int(max),
		params,
	)

	if changed {
		ui.fader.HandleUIChange(int64(newValue))
	}

	// Draw VU meter alongside fader
	if ui.vuMeter != nil {
		imgui.SameLine()
		ui.vuMeter.Draw(state)
	}

	// Draw waterfall beside VU meter when enabled
	if ui.showWaterfall && ui.vuWaterfall != nil {
		imgui.SameLineV(0, 8) // 8px gap between VU meter and waterfall
		ui.vuWaterfall.Draw(state)
	}

	// Draw value display
	valueStr := ui.params.Format(float32(currentValue-min) / float32(max-min))
	imgui.Text(valueStr)

	return columnWidth
}

// getNormalizedLevels returns level values normalized to 0.0-1.0 using dB scale
func (ui *DeviceFaderUI) getNormalizedLevels() []float32 {
	if len(ui.levelControls) == 0 {
		return nil
	}

	levels := make([]float32, len(ui.levelControls))
	for i, ctl := range ui.levelControls {
		val, err := ctl.GetValue()
		if err != nil {
			levels[i] = 0
			continue
		}

		// Apply smoothing if enabled
		if ui.levelBuffers != nil {
			ui.levelBuffers[i].Push(val)
			val = ui.levelBuffers[i].Average()
		}

		// Normalize using dB scale
		if val <= ui.levelMin || ui.levelMax <= 0 {
			levels[i] = 0
			continue
		}

		ratio := float64(val) / float64(ui.levelMax)
		db := 20.0 * math.Log10(ratio)

		const dbRange = 96.0
		if db < -dbRange {
			db = -dbRange
		}
		normalized := float32((db + dbRange) / dbRange)

		if normalized < 0 {
			normalized = 0
		} else if normalized > 1 {
			normalized = 1
		}
		levels[i] = normalized
	}

	return levels
}

// getLevelColor computes track color based on signal level
func (ui *DeviceFaderUI) getLevelColor() *imgui.Vec4 {
	if len(ui.levelControls) == 0 {
		return nil
	}

	// Get max level across all controls
	var maxLevel int64
	for i, ctl := range ui.levelControls {
		val, err := ctl.GetValue()
		if err != nil {
			continue
		}

		if ui.levelBuffers != nil {
			ui.levelBuffers[i].Push(val)
			val = ui.levelBuffers[i].Average()
		}

		if val > maxLevel {
			maxLevel = val
		}
	}

	if maxLevel == 0 {
		return nil
	}

	// Normalize using dB scale
	var normalized float32
	if maxLevel <= ui.levelMin || ui.levelMax <= 0 {
		normalized = 0
	} else {
		ratio := float64(maxLevel) / float64(ui.levelMax)
		db := 20.0 * math.Log10(ratio)

		const dbRange = 96.0
		if db < -dbRange {
			db = -dbRange
		}
		normalized = float32((db + dbRange) / dbRange)
	}

	if normalized <= 0 {
		return nil
	}
	if normalized > 1 {
		normalized = 1
	}

	// HSV color gradient
	var h, s, v float32
	s = 1.0

	if normalized <= 0.5 {
		h = 120.0 / 360.0
		v = 0.3 + (normalized/0.5)*0.3
	} else if normalized <= 0.8 {
		t := (normalized - 0.5) / 0.3
		h = (120.0 - t*60.0) / 360.0
		v = 0.6 + t*0.2
	} else {
		t := (normalized - 0.8) / 0.2
		h = (60.0 - t*60.0) / 360.0
		v = 0.8 + t*0.2
	}

	var r, g, b float32
	imgui.ColorConvertHSVtoRGB(h, s, v, &r, &g, &b)

	return &imgui.Vec4{X: r, Y: g, Z: b, W: 1.0}
}

// GetFader returns the underlying device fader
func (ui *DeviceFaderUI) GetFader() *session.DeviceFader {
	return ui.fader
}

// IsWaterfallOpen returns whether the waterfall is currently shown
func (ui *DeviceFaderUI) IsWaterfallOpen() bool {
	return ui.showWaterfall
}

// GetColumnWidth returns the current column width needed for this fader
func (ui *DeviceFaderUI) GetColumnWidth() float32 {
	faderWidth := float32(60.0)
	vuWidth := float32(0)
	if ui.vuMeter != nil {
		vuWidth = ui.vuMeter.Width() + 8 // 8px gap after fader
	}
	waterfallWidth := float32(0)
	if ui.showWaterfall && ui.vuWaterfall != nil {
		waterfallWidth = ui.vuWaterfall.Width() + 8 // 8px gap between VU and waterfall
	}
	return faderWidth + vuWidth + waterfallWidth + 16 // 16px padding
}
