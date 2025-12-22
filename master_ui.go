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

	// Level meter controls for mix outputs
	levelControls []*scarlettctl.Control

	// Level smoothing buffers
	levelBuffers []*RingBuffer[int64]

	// Level min/max
	levelMin, levelMax int64
}

// NewMasterUI creates a new master UI component for a cue mix.
func NewMasterUI(cueMix *session.CueMix, state *topology.DeviceState, levelSmoothing int) *MasterUI {
	ui := &MasterUI{
		cueMix: cueMix,
		state:  state,
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
		ui.vuMeter.SegmentCount = 141
		ui.vuMeter.SegmentGap = 1
		ui.vuMeter.Height = 301

		// Label channels L/R or M
		labels := make([]string, len(ui.levelControls))
		if len(labels) == 1 {
			labels[0] = "M"
		} else if len(labels) == 2 {
			labels[0] = "L"
			labels[1] = "R"
		}
		ui.vuMeter.SetLabels(labels)
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
	if imgui.ButtonV(fmt.Sprintf("%s##mute_%p", buttonLabel, ui), imgui.Vec2{X: 60, Y: 30}) {
		ui.cueMix.ToggleMute()
	}

	imgui.PopStyleColorV(3)

	// Future: Add more control buttons here (e.g., Solo, PFL, etc.)
}

// DrawOutputVU renders the output VU meters.
func (ui *MasterUI) DrawOutputVU(state *dfx.State) {
	// Update VU meter levels
	if ui.vuMeter != nil {
		levels := ui.getNormalizedLevels()
		ui.vuMeter.SetLevels(levels)
	}

	imgui.Text("Output")

	// Output VU meters
	if ui.vuMeter != nil {
		ui.vuMeter.Draw(state)
	}
}

// GetVUMeterWidth returns the width of the VU meter (for layout calculations)
func (ui *MasterUI) GetVUMeterWidth() float32 {
	if ui.vuMeter != nil {
		return ui.vuMeter.Width()
	}
	return 40 // Default width if no VU meter
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
