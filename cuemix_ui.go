package sessionmixer

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/michaelquigley/dfx"
	"github.com/michaelquigley/sessionmixer/session"
	"github.com/michaelquigley/sessionmixer/topology"
)

// CueMixUI renders a single cue mix in a collapsible section.
type CueMixUI struct {
	// The underlying cue mix
	cueMix *session.CueMix

	// Device fader UIs
	deviceFaders []*DeviceFaderUI

	// Master UI (mute + output VU)
	masterUI *MasterUI

	// UI state
	expanded bool
}

// NewCueMixUI creates a new UI component for a cue mix.
func NewCueMixUI(cueMix *session.CueMix, state *topology.DeviceState, levelSmoothing int) *CueMixUI {
	ui := &CueMixUI{
		cueMix:   cueMix,
		expanded: true, // Start expanded
	}

	// Create device fader UIs
	for _, fader := range cueMix.Faders {
		faderUI := NewDeviceFaderUI(fader, state, levelSmoothing)
		ui.deviceFaders = append(ui.deviceFaders, faderUI)
	}

	// Create master UI
	ui.masterUI = NewMasterUI(cueMix, state, levelSmoothing)

	return ui
}

// Draw renders the cue mix section.
// Layout: [Output VU] | [Controls] | [Device Faders...]
func (ui *CueMixUI) Draw(state *dfx.State) {
	// Collapsible header
	headerFlags := imgui.TreeNodeFlagsDefaultOpen | imgui.TreeNodeFlagsFramed
	if !ui.expanded {
		headerFlags = imgui.TreeNodeFlagsFramed
	}

	// Add mute indicator to header
	headerText := ui.cueMix.Name()
	if ui.cueMix.IsMuted() {
		headerText += " [MUTED]"
	}

	ui.expanded = imgui.CollapsingHeaderTreeNodeFlagsV(
		fmt.Sprintf("%s##cuemix_%p", headerText, ui),
		headerFlags,
	)

	if !ui.expanded {
		return
	}

	// Content area
	contentHeight := float32(380) // Fader height + labels + padding
	imgui.BeginChildStrV(
		fmt.Sprintf("##cuemix_content_%p", ui),
		imgui.Vec2{X: 0, Y: contentHeight},
		imgui.ChildFlagsNone,
		imgui.WindowFlagsHorizontalScrollbar,
	)

	// Get window position for drawing separators
	windowPos := imgui.CursorScreenPos()
	drawList := imgui.WindowDrawList()

	// Calculate column widths with padding for separators
	numFaders := len(ui.deviceFaders)
	faderWidth := float32(100)                    // Base width per device fader column
	controlsWidth := float32(80)                  // Width for controls column
	vuWidth := ui.masterUI.GetVUMeterWidth() + 20 // VU meter width + padding

	// Padding configuration: asymmetric around separators
	// - Right padding (before separator): 25px
	// - Left padding (after separator): 12px (reduced by 50%)
	leftPadding := float32(12)      // Padding before output VU (reduced from 25)
	paddingBeforeSep := float32(25) // Right side padding before separator
	paddingAfterSep := float32(12)  // Left side padding after separator (reduced from 25)
	sectionPadding := paddingBeforeSep + paddingAfterSep

	// Total columns: left padding + output VU + spacer + controls + spacer + device faders
	totalColumns := numFaders + 5
	totalWidth := leftPadding + vuWidth + sectionPadding + controlsWidth + sectionPadding + float32(numFaders)*faderWidth + 50

	// Create a table for layout (no borders - we'll draw our own)
	if imgui.BeginTableV(fmt.Sprintf("##cuemix_table_%p", ui), int32(totalColumns), imgui.TableFlagsNone, imgui.Vec2{X: totalWidth, Y: 0}, 0) {
		// Setup columns: left padding first
		imgui.TableSetupColumnV("##col_left_pad", imgui.TableColumnFlagsWidthFixed, leftPadding, 0)
		// Output VU column
		imgui.TableSetupColumnV("##col_output_vu", imgui.TableColumnFlagsWidthFixed, vuWidth, 0)
		// Spacer column after output VU
		imgui.TableSetupColumnV("##col_spacer1", imgui.TableColumnFlagsWidthFixed, sectionPadding, 0)
		// Controls column
		imgui.TableSetupColumnV("##col_controls", imgui.TableColumnFlagsWidthFixed, controlsWidth, 0)
		// Spacer column after controls
		imgui.TableSetupColumnV("##col_spacer2", imgui.TableColumnFlagsWidthFixed, sectionPadding, 0)
		// Device faders (rightmost)
		for i := range ui.deviceFaders {
			imgui.TableSetupColumnV(fmt.Sprintf("##col_fader_%d", i), imgui.TableColumnFlagsWidthFixed, faderWidth, 0)
		}

		// Draw row
		imgui.TableNextRow()

		// Left padding column
		imgui.TableNextColumn()

		// Output VU column
		imgui.TableNextColumn()
		ui.masterUI.DrawOutputVU(state)

		// Spacer column
		imgui.TableNextColumn()

		// Controls column (MUTE button, etc.)
		imgui.TableNextColumn()
		ui.masterUI.DrawControls(state)

		// Spacer column
		imgui.TableNextColumn()

		// Device faders
		for _, faderUI := range ui.deviceFaders {
			imgui.TableNextColumn()
			faderUI.Draw(state)
		}

		imgui.EndTable()
	}

	// Draw vertical separator lines between sections
	separatorColor := imgui.ColorU32Vec4(imgui.Vec4{X: 0.5, Y: 0.5, Z: 0.5, W: 0.5})
	lineThickness := float32(1.0)

	// Separator after Output VU column (at paddingBeforeSep into the spacer)
	sep1X := windowPos.X + leftPadding + vuWidth + paddingBeforeSep
	drawList.AddLineV(
		imgui.Vec2{X: sep1X, Y: windowPos.Y},
		imgui.Vec2{X: sep1X, Y: windowPos.Y + contentHeight - 20},
		separatorColor,
		lineThickness,
	)

	// Separator after Controls column (at paddingBeforeSep into the spacer)
	sep2X := windowPos.X + leftPadding + vuWidth + sectionPadding + controlsWidth + paddingBeforeSep
	drawList.AddLineV(
		imgui.Vec2{X: sep2X, Y: windowPos.Y},
		imgui.Vec2{X: sep2X, Y: windowPos.Y + contentHeight - 20},
		separatorColor,
		lineThickness,
	)

	imgui.EndChild()
}

// GetFaders returns the device fader UIs
func (ui *CueMixUI) GetFaders() []*DeviceFaderUI {
	return ui.deviceFaders
}

// GetCueMix returns the underlying cue mix
func (ui *CueMixUI) GetCueMix() *session.CueMix {
	return ui.cueMix
}

// IsExpanded returns whether the section is expanded
func (ui *CueMixUI) IsExpanded() bool {
	return ui.expanded
}

// SetExpanded sets the expanded state
func (ui *CueMixUI) SetExpanded(expanded bool) {
	ui.expanded = expanded
}
