# Bidirectional Update Strategy: Hardware-UI Synchronization

This document describes how the ALSA Scarlett GUI application manages bidirectional updates between UI controls and hardware state without using debouncing or complex locking mechanisms.

## Table of Contents

1. [Overview](#overview)
2. [The Core Strategy: Value Equality Checks](#the-core-strategy-value-equality-checks)
3. [Architectural Patterns](#architectural-patterns)
4. [Hardware Monitoring System](#hardware-monitoring-system)
5. [UI to Hardware Updates](#ui-to-hardware-updates)
6. [Feedback Loop Prevention](#feedback-loop-prevention)
7. [Special Cases: Polling and Timers](#special-cases-polling-and-timers)
8. [Code References](#code-references)
9. [Comparison to Debouncing](#comparison-to-debouncing)
10. [Recommendations for Other Projects](#recommendations-for-other-projects)

## Overview

When building applications that allow users to control hardware while simultaneously monitoring hardware state, a common problem arises: bidirectional updates can cause jerkiness, feedback loops, or conflicting states.

**Common Problem:**
```
User moves slider → Updates hardware → Hardware event fired →
Updates UI slider → Triggers change event → Updates hardware → ...
[INFINITE LOOP]
```

**This Application's Solution:**
- **No debouncing** on user input
- **No locking mechanisms**
- **Simple value equality checks** to break feedback loops
- **Hardware as single source of truth**
- **Immediate writes** with natural batching by the driver

## The Core Strategy: Value Equality Checks

The key insight is that when hardware state updates the UI with the same value the user just set, we can detect this and skip emitting change events.

### Implementation

**File:** `src/gtkdial.c` (lines 1294-1327)

```c
static int set_value(GtkDial *dial, double newval) {
  // Clamp and round the value
  double mn = gtk_adjustment_get_lower(dial->adj);
  double mx = gtk_adjustment_get_upper(dial->adj);
  newval = round(newval);
  if (newval < mn) newval = mn;
  if (newval > mx) newval = mx;

  double oldval = gtk_adjustment_get_value(dial->adj);
  double old_peak = dial->current_peak;
  gtk_dial_add_hist_value(dial, newval);

  // CRITICAL: Only emit signal if value actually changed
  if (oldval == newval && old_peak == dial->current_peak)
    return 0;  // Return without emitting signal

  gtk_adjustment_set_value(dial->adj, newval);
  g_signal_emit(dial, signals[VALUE_CHANGED], 0);

  return oldval != newval || old_peak != dial->current_peak;
}
```

### The Update Cycle

```
1. User drags dial
   ↓
2. gain_changed() callback fires
   ↓
3. alsa_set_elem_value() writes to hardware
   ↓
4. Hardware state changes
   ↓
5. ALSA generates event
   ↓
6. alsa_card_callback() receives event
   ↓
7. gain_updated() callback fires
   ↓
8. gtk_dial_set_value() called with same value
   ↓
9. set_value() detects oldval == newval
   ↓
10. Returns 0 without emitting VALUE_CHANGED signal
   ↓
11. Feedback loop broken!
```

## Architectural Patterns

### Pattern 1: Single Source of Truth

**Hardware is authoritative.** The application never maintains a separate copy of control values (except for simulated cards in testing).

**File:** `src/alsa.c` (lines 246-280)

```c
void alsa_set_elem_value(struct alsa_elem *elem, long value) {
  // For real hardware (not simulation):
  snd_ctl_elem_value_alloca(&elem_value);
  snd_ctl_elem_value_set_numid(elem_value, elem->numid);

  // Read current value from hardware first
  snd_ctl_elem_read(elem->card->handle, elem_value);

  // Set new value
  if (type == SND_CTL_ELEM_TYPE_INTEGER) {
    snd_ctl_elem_value_set_integer(elem_value, elem->index, value);
  }
  // ... (other types)

  // Write directly to hardware
  snd_ctl_elem_write(elem->card->handle, elem_value);
}
```

No intermediate application state that could diverge from reality.

### Pattern 2: Unidirectional Data Flow

```
┌──────────────┐
│  User Input  │
└──────┬───────┘
       ↓
┌──────────────────┐
│  Write Hardware  │
└──────┬───────────┘
       ↓
┌──────────────────┐
│  Hardware Event  │
└──────┬───────────┘
       ↓
┌──────────────┐
│  Update UI   │
└──────────────┘
```

Data always flows in one direction: UI → Hardware → UI

### Pattern 3: Callback Registration System

Each widget registers a callback with its associated ALSA element to receive hardware change notifications.

**File:** `src/alsa.h` (lines 18-20)

```c
typedef void (AlsaElemCallback)(struct alsa_elem *, void *);
```

**File:** `src/widget-gain.c` (lines 243-247)

```c
// Register UI change handler
g_signal_connect(
  data->dial, "value-changed", G_CALLBACK(gain_changed), data
);

// Register hardware change handler
alsa_elem_add_callback(elem, gain_updated, data);
```

### Pattern 4: Widget Sensitivity Control

Widgets are disabled when hardware is not writable, preventing conflicting input.

**File:** `src/widget-gain.c` (lines 46-98)

```c
static void gain_updated(struct alsa_elem *elem, void *private) {
  struct gain *data = private;

  // Check if hardware control is writable
  int is_writable = alsa_get_elem_writable(elem);
  gtk_widget_set_sensitive(data->dial, is_writable);

  // Update UI to match hardware
  int alsa_value = alsa_get_elem_value(elem);
  gtk_dial_set_value(GTK_DIAL(data->dial), alsa_value);

  // Update label...
}
```

## Hardware Monitoring System

### Event-Driven Architecture Using GIO Channels

The application uses ALSA's native event subscription system integrated with GTK's main loop.

**File:** `src/alsa.c` (lines 966-983)

```c
static void alsa_add_card_callback(struct alsa_card *card) {
  // Create GIO channel from ALSA file descriptor
  card->io_channel = g_io_channel_unix_new(card->pfd.fd);

  // Add to GTK main loop
  card->event_source_id = g_io_add_watch_full(
    card->io_channel,
    0,
    G_IO_IN | G_IO_ERR | G_IO_HUP,
    alsa_card_callback,
    card,
    card_destroy_callback
  );
}

static void alsa_subscribe(struct alsa_card *card) {
  // Subscribe to hardware events
  snd_ctl_subscribe_events(card->handle, 1);
  snd_ctl_poll_descriptors(card->handle, &card->pfd, 1);
}
```

### Hardware Event Callback

**File:** `src/alsa.c` (lines 866-920)

```c
static gboolean alsa_card_callback(
  GIOChannel    *source,
  GIOCondition   condition,
  void          *data
) {
  struct alsa_card *card = data;
  snd_ctl_event_t *event;

  snd_ctl_event_alloca(&event);
  int err = snd_ctl_read(card->handle, event);

  unsigned int mask = snd_ctl_event_get_mask(event);

  // Only process value or info changes
  if (!(mask & (SND_CTL_EVENT_MASK_VALUE | SND_CTL_EVENT_MASK_INFO)))
    return 1;

  unsigned int numid = snd_ctl_event_elem_get_numid(event);

  // Find the element that changed and call its callbacks
  for (int i = 0; i < card->elems->len; i++) {
    struct alsa_elem *elem = &g_array_index(
      card->elems, struct alsa_elem, i
    );
    if (elem->numid == numid) {
      alsa_elem_change(elem);
      break;
    }
  }

  return 1;  // Keep event source active
}
```

### Callback Dispatch

**File:** `src/alsa.c` (lines 760-772)

```c
static void alsa_elem_change(struct alsa_elem *elem) {
  if (!elem || !elem->callbacks)
    return;

  // Call all registered callbacks for this element
  for (GList *l = elem->callbacks; l; l = l->next) {
    struct alsa_elem_callback *cb =
      (struct alsa_elem_callback *)l->data;
    if (!cb || !cb->callback)
      continue;
    cb->callback(elem, cb->data);
  }
}
```

**Key Points:**
- No polling for regular controls
- Hardware events trigger callbacks only for changed elements
- Integrates seamlessly with GTK's event loop
- Single-threaded, no race conditions

## UI to Hardware Updates

### Example: Gain Widget

**File:** `src/widget-gain.c`

**User Input Handler (lines 20-44):**
```c
static void gain_changed(GtkWidget *widget, struct gain *data) {
  // Get value from UI control
  int value = gtk_dial_get_value(GTK_DIAL(data->dial));

  // Write directly to hardware - no batching, no delay
  alsa_set_elem_value(data->elem, value);

  // Optional: Update related controls (e.g., Direct Monitor Mix)
  if (data->dmx_elem) {
    int dmx_value = alsa_get_elem_value(data->dmx_elem);
    if (dmx_value) {
      int dmx_scaled_value = ((value * dmx_value) + 127) / 255;
      alsa_set_elem_value(data->dmx_out_elem, dmx_scaled_value);
    }
  }
}
```

**Hardware Update Handler (lines 46-98):**
```c
static void gain_updated(struct alsa_elem *elem, void *private) {
  struct gain *data = private;

  // Enable/disable based on hardware state
  int is_writable = alsa_get_elem_writable(elem);
  gtk_widget_set_sensitive(data->dial, is_writable);

  // Update dial to match hardware
  int alsa_value = alsa_get_elem_value(elem);
  gtk_dial_set_value(GTK_DIAL(data->dial), alsa_value);

  // Update label with dB value
  double db = alsa_value / 256.0;
  snprintf(s, 20, "%.2fdB", db);
  gtk_label_set_text(GTK_LABEL(data->label), s);

  // Update related controls...
}
```

### Other Widget Types

All widgets follow the same pattern:

**Boolean (Toggle Buttons)** - `src/widget-boolean.c` (lines 16-19, 34-47)
```c
static void toggle_button_changed(GtkWidget *widget, void *user_data) {
  struct toggle_button *data = user_data;
  int value = gtk_toggle_button_get_active(GTK_TOGGLE_BUTTON(widget));
  alsa_set_elem_value(data->elem, value);
}

static void update_toggle_button(void *user_data) {
  struct toggle_button *data = user_data;
  int value = alsa_get_elem_value(data->elem);
  gtk_toggle_button_set_active(GTK_TOGGLE_BUTTON(data->toggle), value);
}
```

**Drop-down (Enums)** - `src/widget-drop-down.c` (lines 38-46, 102-124)
```c
static void drop_down_changed(GtkWidget *widget, void *user_data) {
  struct drop_down *data = user_data;
  int value = gtk_combo_box_get_active(GTK_COMBO_BOX(data->drop_down));
  alsa_set_elem_value(data->elem, value);
}

static void update_drop_down(void *user_data) {
  struct drop_down *data = user_data;
  int value = alsa_get_elem_value(data->elem);
  gtk_combo_box_set_active(GTK_COMBO_BOX(data->drop_down), value);
}
```

## Feedback Loop Prevention

### Why Feedback Loops Don't Happen

1. **User drags dial** → Gesture update callback fires repeatedly during drag
2. **Each mouse move** → `gain_changed()` → writes new value to hardware
3. **Hardware updates** → ALSA event generated
4. **ALSA event** → `gain_updated()` → calls `gtk_dial_set_value()`
5. **`gtk_dial_set_value()`** → calls internal `set_value()`
6. **`set_value()` checks** → `oldval == newval`?
7. **If equal** → Return 0, **DO NOT** emit `VALUE_CHANGED` signal
8. **No signal** → `gain_changed()` not called again
9. **Loop broken!**

### Drag Gesture Implementation

**File:** `src/gtkdial.c` (lines 1487-1544)

```c
static void gtk_dial_drag_gesture_update(
  GtkGestureDrag *gesture,
  double          offset_x,
  double          offset_y,
  GtkDial        *dial
) {
  // Calculate new value based on drag offset
  double DRAG_FACTOR = 1.5;
  double valp = dial->dvalp - DRAG_FACTOR * (offset_y / dial->h);

  // Immediately write on EVERY mouse move event
  // No batching, no debouncing
  set_value(dial, calc_val(valp, mn, mx));

  // Queue redraw (GTK coalesces rapid redraws automatically)
  gtk_widget_queue_draw(GTK_WIDGET(dial));
}
```

**Why This Works Without Jerkiness:**
1. **ALSA driver** batches rapid updates internally
2. **Value equality check** prevents redundant signal emissions
3. **GTK's draw queue** naturally coalesces rapid redraws
4. **Small deadband** (1px) in drag calculation prevents micro-changes
5. **Single-threaded** event loop eliminates race conditions

## Special Cases: Polling and Timers

### When Polling IS Used

Not all controls can use event-driven updates. Some require polling:

#### 1. Volatile Controls

**File:** `src/widget-boolean.c` (lines 121-123)

```c
// Periodically update volatile controls
if (alsa_get_elem_volatile(elem)) {
  data->source = g_timeout_add_seconds(
    1,
    (GSourceFunc)update_toggle_button,
    data
  );
}
```

**Volatile controls** are those that can change without generating ALSA events (e.g., hardware-controlled indicators). They're polled every 1 second.

#### 2. Level Meters

**File:** `src/window-levels.c` (lines 36-53, 167)

```c
static int update_levels_controls(void *user_data) {
  struct levels *data = user_data;
  struct alsa_elem *level_meter_elem = data->level_meter_elem;

  // Read all meter values from hardware
  long *values = alsa_get_elem_int_values(level_meter_elem);

  // Update peak indicators
  gtk_dial_peak_tick();

  // Update each meter display
  for (int i = 0; i < level_meter_elem->count; i++) {
    // Convert to dB
    double value = 20 * log10(values[i] / 4095.0);
    gtk_dial_set_value(GTK_DIAL(data->meters[i]), value);
  }

  free(values);
  return 1;  // Continue timer
}

// Install timer: 50ms interval = 20Hz update rate
data->timer = g_timeout_add(50, update_levels_controls, data);
```

**Why polling is necessary for meters:**
- They update very frequently (20Hz)
- They're read-only displays (`can_control` flag is FALSE)
- No user interaction possible, so no conflict
- ALSA doesn't generate events for every meter update (too frequent)

### Event-Driven vs Polling Decision Tree

```
Is control read-only (display only)?
├─ Yes: Use polling (e.g., level meters)
│   └─ Frequency: 50ms for real-time displays, 1s for indicators
│
└─ No: Can control change without ALSA event?
    ├─ Yes: Use polling (1 second timer)
    │   └─ Example: Volatile controls
    │
    └─ No: Use event-driven updates
        └─ Example: Regular controls (gain, mute, etc.)
```

## Code References

### Core System

| Component | File | Lines | Description |
|-----------|------|-------|-------------|
| Hardware event subscription | `src/alsa.c` | 966-983 | GIO channel setup for ALSA events |
| Event callback dispatcher | `src/alsa.c` | 866-920 | Main hardware event handler |
| Element callback list | `src/alsa.c` | 760-772 | Triggers widget updates |
| Direct hardware write | `src/alsa.c` | 246-280 | ALSA element write function |
| Callback registration | `src/alsa.c` | 774-797 | Add/remove callbacks for elements |

### Widgets (UI ↔ Hardware)

| Widget Type | File | UI→HW Lines | HW→UI Lines | Description |
|-------------|------|-------------|-------------|-------------|
| Gain (dial) | `src/widget-gain.c` | 20-44 | 46-98 | Rotary controls with dB display |
| Boolean (toggle) | `src/widget-boolean.c` | 16-19 | 34-47 | On/off switches |
| Drop-down (enum) | `src/widget-drop-down.c` | 38-46 | 102-124 | Enumerated choice controls |
| Air/phantom switches | `src/widget-air-phantom.c` | 16-24 | 26-51 | Special boolean controls |
| Matrix routing | `src/window-routing.c` | 180-197 | 87-120 | Routing matrix buttons |

### Feedback Prevention

| Component | File | Lines | Description |
|-----------|------|-------|-------------|
| Value change detection | `src/gtkdial.c` | 1294-1327 | Core feedback loop prevention |
| Drag gesture handling | `src/gtkdial.c` | 1487-1544 | Real-time user input without debouncing |
| Toggle button equality | GTK built-in | N/A | GTK automatically prevents redundant toggles |
| Combo box equality | GTK built-in | N/A | GTK automatically prevents redundant selections |

### Polling Mechanisms

| Component | File | Lines | Description |
|-----------|------|-------|-------------|
| Volatile control polling | `src/widget-boolean.c` | 121-123 | 1-second timer for volatile controls |
| Level meter polling | `src/window-levels.c` | 36-53, 167 | 50ms timer for level meters |
| Peak tick | `src/gtkdial.c` | 1098-1112 | Peak hold decay mechanism |

## Comparison to Debouncing

### Debouncing Approach

**Typical implementation:**
```javascript
let timeout;
function onSliderChange(value) {
  clearTimeout(timeout);
  timeout = setTimeout(() => {
    writeToHardware(value);
  }, 100);  // Wait 100ms after user stops changing
}
```

**Pros:**
- Reduces number of hardware writes
- Simple to implement
- Works with any hardware

**Cons:**
- Adds latency (delay before hardware updates)
- Feels sluggish during rapid changes
- Still needs mechanism to prevent hardware→UI→hardware loops
- User sees UI change before hardware responds

### This Application's Approach

**Implementation:**
```c
void onDialChange(value) {
  writeToHardware(value);  // Immediate write, no delay
}

void onHardwareChange(value) {
  if (getCurrentValue() == value)
    return;  // Skip if value unchanged
  updateUI(value);
  emitSignal();
}
```

**Pros:**
- Zero latency - immediate hardware response
- Feels responsive during rapid changes
- Simple logic - just value comparison
- Hardware driver handles write coalescing naturally
- No artificial delays

**Cons:**
- More hardware writes (but driver batches them)
- Requires value equality check in widget implementation
- May not work well with imprecise floating-point values

### Performance Comparison

| Metric | Debouncing | Value Equality |
|--------|-----------|----------------|
| User input latency | 100-300ms | 0ms |
| Hardware writes per drag | ~5-10 | ~50-100 |
| Perceived responsiveness | Sluggish | Immediate |
| Code complexity | Low | Low |
| Hardware load | Low | Medium (driver optimized) |
| Feedback loop prevention | Additional logic needed | Built-in |

## Recommendations for Other Projects

### When to Use This Approach

**Good fit:**
- Hardware with fast response times
- Driver that can handle rapid updates
- Integer or discrete values (easy equality checks)
- Single-threaded UI framework
- Need for immediate user feedback

**Not recommended:**
- Slow hardware (>50ms response time)
- Floating-point values without rounding
- Multi-threaded architecture (needs locking)
- Hardware with rate limiting or command queues
- Network-based hardware with high latency

### Implementation Checklist

1. **Make hardware the source of truth**
   - Don't maintain separate application state
   - UI should always reflect hardware

2. **Implement value equality checks**
   - In every `setValue()` method
   - Compare old and new values
   - Skip signal emission if unchanged

3. **Use event-driven updates when possible**
   - Subscribe to hardware change notifications
   - Only poll for truly volatile controls

4. **Write to hardware immediately**
   - Don't batch user input
   - Let the driver/kernel handle optimization

5. **Disable controls during conflicts**
   - Check if hardware is writable
   - Disable UI controls when appropriate

6. **Add small deadband for continuous controls**
   - Prevent micro-changes from drag jitter
   - 1-2 pixel/unit deadband is sufficient

7. **Use framework's built-in coalescing**
   - GTK/Qt automatically batch redraws
   - Don't manually throttle UI updates

### Example Pseudocode

```c
// Widget class
class HardwareSlider {
  void onUserDrag(int newValue) {
    // Immediate write, no debouncing
    hardware.write(element, newValue);
  }

  void onHardwareChange(int newValue) {
    // Value equality check prevents feedback loop
    int oldValue = slider.getValue();
    if (oldValue == newValue)
      return;  // Don't emit change signal

    slider.setValue(newValue);
    emit valueChanged(newValue);
  }

  void setup() {
    // Register both directions
    slider.onChanged -> onUserDrag
    hardware.subscribe(element, onHardwareChange)
  }
}
```

### Adapting to Different Technologies

**Web (JavaScript):**
```javascript
class HardwareControl {
  setValue(newValue) {
    const oldValue = this.currentValue;
    this.currentValue = newValue;
    this.element.value = newValue;

    // Value equality check
    if (oldValue !== newValue) {
      this.emit('change', newValue);
    }
  }

  onUserInput(event) {
    const value = parseInt(event.target.value);
    this.hardware.write(this.controlId, value);
  }

  onHardwareUpdate(value) {
    this.setValue(value);  // Equality check inside
  }
}
```

**Python (PyQt):**
```python
class HardwareSlider(QSlider):
    def setValue(self, value):
        old_value = self.value()
        super().setValue(value)

        # Only emit if changed
        if old_value != value:
            self.valueChanged.emit(value)

    def on_user_change(self, value):
        # Immediate write
        self.hardware.write(self.element_id, value)

    def on_hardware_change(self, value):
        # setValue includes equality check
        self.setValue(value)
```

### Common Pitfalls to Avoid

1. **Don't maintain duplicate state**
   - Bad: `app_state[id] = value; hardware.write(value);`
   - Good: `hardware.write(value);` (hardware is the state)

2. **Don't debounce hardware→UI updates**
   - These should be immediate
   - Only the value equality check should filter them

3. **Don't use `===` for floating-point**
   - Round values first: `Math.round(value * 100) / 100`
   - Or use epsilon comparison: `Math.abs(a - b) < 0.01`

4. **Don't block the UI thread**
   - Hardware writes should be async or very fast
   - If slow, show intermediate state

5. **Don't skip writable checks**
   - Always check if hardware control is writable
   - Disable UI when not writable

## Conclusion

The ALSA Scarlett GUI demonstrates that debouncing is not always necessary for smooth bidirectional hardware-UI synchronization. By using:

- **Value equality checks** to break feedback loops
- **Immediate hardware writes** for responsive feel
- **Hardware as source of truth** for consistency
- **Event-driven updates** for efficiency
- **Simple unidirectional data flow** for predictability

The application achieves smooth, responsive control without complex synchronization mechanisms. This approach is particularly effective for hardware with fast response times and drivers that can optimize rapid updates.

The key insight: **Let the hardware driver handle write optimization, focus your application logic on preventing feedback loops through value comparison.**
