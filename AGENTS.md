# sessionmixer

A desktop mixer control surface for audio interfaces with onboard DSP mixing, built with Go and dfx: named devices over the hardware's ports, stereo cue mixes over its mix buses, mute as routing, bidirectional sync with the hardware, and level metering. Today that means Focusrite Scarlett 4th-generation interfaces through `scarlettctl`. A personal studio tool, published as open source.

## Status

The mixer is built and in daily use: device and cue mix configuration, stereo ganging, mute as routing, non-destructive startup, VU meters and the waterfall, and bidirectional sync. `docs/current/` is the record of what exists and how it behaves. No release has been tagged; `CHANGELOG.md` accumulates under `## Unreleased`. What is ahead is on the roadmap.

## Stack

- Go, single module `github.com/michaelquigley/sessionmixer`.
- cgo, through `scarlettctl`, against libasound (ALSA). The hardware is reached only through `scarlettctl.Card` and `scarlettctl.Control`.
- `github.com/spf13/cobra` with `charmbracelet/fang` for the CLI. `github.com/michaelquigley/df/dl` for logging, `github.com/michaelquigley/df/dd` for configuration, `github.com/pkg/errors` for wrapping.
- dfx (`github.com/michaelquigley/dfx`) for the ui, pinned at a tag; `scarlettctl` pinned at a commit.
- `go.mod` carries no `replace` directives. The module builds from published versions, because CI and patchbay (which imports `sessionmixer/topology`) consume it from GitHub. To develop dfx or scarlettctl in lockstep, use a `go.work` (ignored by git), and bump the pin once the dependency is tagged or pushed.

## Building

- `make` installs, `make test` runs tests plus `go vet`, `make clean` clears the project GOBIN.
- The machine needs `libasound2-dev`, and the X11 and OpenGL development packages dfx's GLFW backend links against.
- `sessionmixer run` opens the window against the card named in `~/.config/sessionmixer/session.yaml` (`etc/example-config.yaml` is the shape). `sessionmixer topology` prints what the card exposes. Nothing in `go test` touches hardware.

## Layout

- `cmd/sessionmixer/` — cobra entry: `run` opens the window; `topology` prints the discovered device (`--routing`, `--mixes`, `--meters`); `desktop integrate` and `desktop remove` install and remove the Linux launcher entry and icon for the running binary.
- `session/` — the user-facing model: `SessionConfig`, `Device`, `CueMix`, `DeviceFader`, the builder that resolves configuration against the topology, mute and routing (`cuemix.go`), and the bidirectional fader (`device_fader.go`).
- `topology/` — the hardware model: `Port`, `Mix`, `MixInput`, `RoutingEndpoint`, `Device`, `DeviceState`; `builder.go` discovers a device from a profile; `detect.go` picks the profile by card name and firmware. `topology/profiles/` holds one profile per device and firmware variant, registered in `init()`.
- The root package is the ui: `mixer.go` (`SessionMixer`), `cuemix_ui.go`, `device_fader_ui.go`, `master_ui.go`, `session_monitor.go` (ALSA events into faders), and the shared meter and ring buffer pieces.
- `docs/current/` — built behavior, including the per-device control layouts the profiles were written from. `docs/future/roadmap/` — the roadmap. `docs/journal/` — project memory.

## Key conventions

- **Value equality breaks the feedback loop.** A ui write reaches the hardware, the hardware fires an event with the same value, and the handler returns early because old equals new. No debouncing, no locks. `docs/current/BIDIRECTIONAL_UPDATE_STRATEGY.md` is the full account; keep the equality checks when touching `DeviceFader.HandleUIChange` or `HandleHWChange`.
- **Mute is routing, and the endpoint cache follows it.** Muting saves each output's source and routes it to `Off`; unmuting restores. Both must update the endpoint's cached `currentSourceIndex`, or the next cycle misbehaves.
- **Startup never writes routing.** A cue mix whose outputs already carry its mixes starts active; otherwise it starts muted with the expected routing saved for unmute. The hardware is read, not modified.
- **Stereo is ganged.** A stereo cue mix (`A+B`) with a stereo device drives Mix A's input for the left port and Mix B's for the right from one fader; a mono device drives the same input in both mixes.
- **Outputs accept device names.** A cue mix's `outputs` may name raw ports or a device; a device name wins a naming conflict.
- **Profiles are keyed by firmware.** Several profiles can describe one card at different firmware versions; add a variant as a new file in `topology/profiles/` registered with its firmware in `init()`.
- **The desktop id is one string.** `desktopID` in `cmd/sessionmixer/desktop.go` names the entry and the icon, and `run.go` passes the same constant as dfx's `AppID`, so the desktop pairs the running window with its entry.
- Comments are lowercase except doc comments, which lead with the identifier.
- **Changelog.** `CHANGELOG.md` follows the in-house convention: newest-first releases, prose entries each led by one of `FEATURE`/`CHANGE`/`FIX`, and an `## Unreleased` slot at the top that agents write into. The full spec is the grimoire's `software/conventions/changelog-convention.md`.

## Review

A change is gated by terminus over the working tree, against sessionmixer's canon (`terminus-canon/projects/sessionmixer`): run it before handing off, present each finding for Michael's decision (apply, adjust, or veto), and bring it to clean. A vetoed finding may be raised again; report it as vetoed. Full-repo reviews are maintenance, run only on request. Their findings are filed as roadmap cards in `inbox`, not fixed in passing. Until the canon project exists, say so in the handoff rather than skipping silently.

## Roadmap

This repo's roadmap lives in `docs/future/roadmap/` — one frontmatter-markdown item per file, per the roadmap convention in the grimoire (software/conventions/roadmap-convention.md). You may add items freely: write the file directly with required `title`, `state: inbox`, and `created:` (today, YYYY-MM-DD), optional `tags`/`source`/`log`, and a body that is a small, clear prompt -- the problem or solution to execute, not documentation of it; trust the code and the day's journal entry for what's discoverable, and point a `log:` stamp at the specific journal entry when a card leans on hard-won context. Everything above the first `##` heading is the prompt; supporting material that isn't the prompt goes in named sections below it (`## why` for justification, `## background` for a longer description), which are conventional, never required, and never validated. The filename is the slug of the title (lowercase ASCII, hyphens; discard every other character); never overwrite an existing file. Read sibling items for the shape.

Hard rules: never touch `order.yaml` (priority is the operator's judgment, set at triage); never commit roadmap changes unless directed — the uncommitted diff is the review queue; never delete items; edits change only the lines that express them. Label the kind from the house set when one fits: defect, documentation, enhancement, epic, feature, story; add `spike` alongside it when the work carries unknowns that need discovery.

## For agents arriving cold

This is `sessionmixer`, part of Michael Quigley's broader software practice. Related projects (`dfx`, `scarlettctl`, `patchbay`, `mercurius`, `terminus`) live as siblings in `~/Repos/q/products/`; patchbay is the other consumer of `topology`, so a change to that package's exported surface is a change to patchbay. The grimoire at `~/Repos/q/writing/grimoire` holds cross-project conventions; for sessionmixer, the in-repo `docs/` is authoritative. If you've been assigned a role, read the grimoire's `agents/agent-roles.md`.

## Project memory

Durable knowledge about this project lives in `docs/journal/`, dated files `docs/journal/YYYY-MM-DD.md`. This is project memory; it does not go in harness-local storage (`.claude/` or equivalent), where it's invisible to every other harness and collaborator and dies with the host. Concretely: do not write to your harness's memory directory or memory tool for this project — even when the harness presents it as the default place for durable knowledge. That tool is the silo this convention exists to replace; the journal is the only durable home.

On arrival, read the most recent entries to pick up where the last session left off, before you start changing things. Treat them as prior-session context, not verified truth — if an entry conflicts with the code or a `docs/current/` doc, the code wins.

Write the smallest entry that carries the session's durable insight, and nothing more. The test for every line: *would a competent agent get this wrong, or waste time rediscovering it, working from the tree alone?* If it's recoverable by reading the code, the diff, `docs/current/`, or git history, leave it out.

That filter keeps four kinds of thing and discards the rest:

- **Decisions whose rationale isn't visible in the result** — why a value was chosen, what a line guards against, why something that looks like dead code or a no-op is load-bearing.
- **Deliberate non-actions** — a change you considered and chose not to make, so the next agent doesn't "fix" it. An unchanged file leaves no trace in a diff.
- **Couplings that span files** — two places that must move together, an ordering that matters, an assumption one file makes about another.
- **Live state** — what's unverified, unfinished, or waiting on something external.

Skip change inventories, restatements of the diff, and play-by-play of how you worked. There's no write-time approval gate; Michael reviews on commit. Append to the day's file if it exists, and write the few lines you'd want the next agent to read — honest and self-contained.

## Commits

The operator commits; agents don't. Never run `git commit` or `git push` in this repo. Finish the edit, leave the change in the working tree (staged is fine), report what changed, and hand off — the uncommitted diff is the review queue and the commit is the operator's act of acceptance. Approval of a change is not direction to commit; only an explicit instruction to commit is, and only for that commit.
