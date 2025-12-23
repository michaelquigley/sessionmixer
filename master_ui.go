package sessionmixer

import (
	"fmt"
	"math"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/michaelquigley/dfx"
	"github.com/michaelquigley/dfx/fonts"
	"github.com/michaelquigley/scarlettctl"
	"github.com/michaelquigley/sessionmixer/session"
	"github.com/michaelquigley/sessionmixer/topology"
)

// MasterUI renders the master section for a cue mix with mute button and output VU meters.
type MasterUI struct {
	// The parent cue mix
	cueMix *session.CueMix

	// Device state for level meter access
	state *topology.DeviceState

	// VU meter for mix output levels
	vuMeter *dfx.VUMeter

	// VU waterfall for level history (optional, toggled by click)
	vuWaterfall *dfx.VUWaterfall

	// Whether waterfall is currently shown
	showWaterfall bool

	// Level meter controls for mix outputs
	levelControls []*scarlettctl.Control

	// Level smoothing buffers
	levelBuffers []*RingBuffer[int64]

	// Level min/max
	levelMin, levelMax int64
}

// NewMasterUI creates a new master UI component for a cue mix.
func NewMasterUI(cueMix *session.CueMix, state *topology.DeviceState, levelSmoothing int, showWaterfall bool) *MasterUI {
	ui := &MasterUI{
		cueMix:        cueMix,
		state:         state,
		showWaterfall: showWaterfall,
	}

	// Collect level meter controls from mix outputs
	if cueMix.MixL != nil && cueMix.MixL.OutputLevelMeterControl != nil {
		ui.levelControls = append(ui.levelControls, cueMix.MixL.OutputLevelMeterControl)
	}
	if cueMix.MixR != nil && cueMix.MixR.OutputLevelMeterControl != nil {
		ui.levelControls = append(ui.levelControls, cueMix.MixR.OutputLevelMeterControl)
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

	// Create VU meter for mix outputs
	if len(ui.levelControls) > 0 {
		ui.vuMeter = dfx.NewVUMeter(len(ui.levelControls))
		ui.vuMeter.Height = 303
		ui.vuMeter.Mode = dfx.VUMeterHighres

		// Label channels L/R or M
		labels := make([]string, len(ui.levelControls))
		if len(labels) == 1 {
			labels[0] = "M"
		} else if len(labels) == 2 {
			labels[0] = "L"
			labels[1] = "R"
		}
		ui.vuMeter.SetLabels(labels)

		// Create VU waterfall (same channel count, matching height)
		ui.vuWaterfall = dfx.NewVUWaterfall(len(ui.levelControls))
		ui.vuWaterfall.Height = 303
		ui.vuWaterfall.ChannelWidth = 12
		ui.vuWaterfall.ChannelGap = 4
		ui.vuWaterfall.RowHeight = 1
		ui.vuWaterfall.Highres = true
		ui.vuWaterfall.SetHistorySize(303)
	}

	return ui
}

// DrawControls renders the control buttons (MUTE, etc.) in a vertical stack.
func (ui *MasterUI) DrawControls(state *dfx.State) {
	imgui.Text("Controls")

	imgui.Dummy(imgui.Vec2{X: 0, Y: 10})

	// Mute button
	muted := ui.cueMix.IsMuted()
	var buttonColor imgui.Vec4
	if muted {
		// Red when muted
		buttonColor = imgui.Vec4{X: 0.8, Y: 0.2, Z: 0.2, W: 1.0}
	} else {
		// Green when not muted
		buttonColor = imgui.Vec4{X: 0.2, Y: 0.6, Z: 0.2, W: 1.0}
	}

	imgui.PushStyleColorVec4(imgui.ColButton, buttonColor)
	imgui.PushStyleColorVec4(imgui.ColButtonHovered, imgui.Vec4{
		X: buttonColor.X + 0.1,
		Y: buttonColor.Y + 0.1,
		Z: buttonColor.Z + 0.1,
		W: 1.0,
	})
	imgui.PushStyleColorVec4(imgui.ColButtonActive, imgui.Vec4{
		X: buttonColor.X + 0.2,
		Y: buttonColor.Y + 0.2,
		Z: buttonColor.Z + 0.2,
		W: 1.0,
	})

	buttonLabel := fonts.ICON_VOLUME_MUTE
	if muted {
		buttonLabel = fonts.ICON_VOLUME_OFF
	}
	if imgui.ButtonV(fmt.Sprintf("%s##mute_%p", buttonLabel, ui), imgui.Vec2{X: 60, Y: 30}) {
		ui.cueMix.ToggleMute()
	}

	// Tooltip for mute button
	if imgui.IsItemHovered() {
		if muted {
			imgui.SetTooltip("Click to unmute output")
		} else {
			imgui.SetTooltip("Click to mute output")
		}
	}

	imgui.PopStyleColorV(3)

	// Future: Add more control buttons here (e.g., Solo, PFL, etc.)
}

// DrawOutputVU renders the output VU meters.
func (ui *MasterUI) DrawOutputVU(state *dfx.State) {
	// Update VU meter levels
	levels := ui.getNormalizedLevels()
	if ui.vuMeter != nil {
		ui.vuMeter.SetLevels(levels)
	}
	if ui.vuWaterfall != nil {
		ui.vuWaterfall.SetLevels(levels)
	}

	// Display configured outputs (click to toggle waterfall)
	// Check if outputs match a device definition
	if deviceName := ui.findMatchingDeviceName(); deviceName != "" {
		imgui.Text(deviceName)
		if imgui.IsItemClicked() {
			ui.showWaterfall = !ui.showWaterfall
		}
	} else {
		// Fall back to individual port short names
		for i, port := range ui.cueMix.Outputs {
			imgui.Text(port.ShortName)
			// Only first label is clickable to toggle
			if i == 0 && imgui.IsItemClicked() {
				ui.showWaterfall = !ui.showWaterfall
			}
		}
	}

	// Output VU meters
	if ui.vuMeter != nil {
		ui.vuMeter.Draw(state)
	}

	// Draw waterfall beside VU meter when enabled
	if ui.showWaterfall && ui.vuWaterfall != nil {
		imgui.SameLineV(0, 8) // 8px gap between VU meter and waterfall
		ui.vuWaterfall.Draw(state)
	}
}

// findMatchingDeviceName checks if the cue mix outputs match a device's ports
// and returns the device name if found, empty string otherwise.
func (ui *MasterUI) findMatchingDeviceName() string {
	if ui.cueMix.Session == nil || len(ui.cueMix.Outputs) == 0 {
		return ""
	}

	for _, device := range ui.cueMix.Session.DeviceList {
		if len(device.Ports) != len(ui.cueMix.Outputs) {
			continue
		}

		match := true
		for i, port := range device.Ports {
			if port != ui.cueMix.Outputs[i] {
				match = false
				break
			}
		}

		if match {
			return device.Name()
		}
	}

	return ""
}

// IsWaterfallOpen returns whether the waterfall is currently shown
func (ui *MasterUI) IsWaterfallOpen() bool {
	return ui.showWaterfall
}

// GetVUMeterWidth returns the width of the VU meter section (for layout calculations)
// Includes waterfall width when shown
func (ui *MasterUI) GetVUMeterWidth() float32 {
	width := float32(40) // Default width if no VU meter
	if ui.vuMeter != nil {
		width = ui.vuMeter.Width()
	}
	if ui.showWaterfall && ui.vuWaterfall != nil {
		width += 8 + ui.vuWaterfall.Width() // 8px gap between VU and waterfall
	}
	return width
}

// getNormalizedLevels returns level values normalized to 0.0-1.0 using dB scale
func (ui *MasterUI) getNormalizedLevels() []float32 {
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
