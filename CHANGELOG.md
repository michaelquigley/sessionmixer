# CHANGELOG

## Unreleased

FEATURE: `make audition` packages the working tree for Ubuntu 24.04 amd64 studio machines using a Podman build, with source identification and an executable-load check. `AUDITION_SOURCE` selects a separate checkout for baseline comparisons.

FIX: Window resizing no longer waits for level-meter hardware reads. Meters share a background sampler across cue mixes, and redraws use cached levels.

## v0.1.0

CHANGE: dfx is pinned at `v0.1.18`, which makes Faceplate the default style: Chakra Petch text, roomier spacing, and charcoal surfaces with a yellow accent.

FIX: Under Faceplate, a cue mix's column labels share one baseline again; the device labels no longer sit lower than the output and Controls labels. The Controls column is centered between its separators, and the separators line up with the columns they divide.

FEATURE: `sessionmixer desktop integrate` installs a Linux desktop entry and the sessionmixer mark for the binary that runs it, so sessionmixer launches from the application grid under its own icon; `sessionmixer desktop remove` takes them out again. The window reports `sessionmixer` as its app id, so the desktop pairs a running window with the entry. This replaces the hand-maintained `sessionmixer.desktop` and `sessionmixer.svg` that lived at the repository root.

CHANGE: The module builds from published dependency versions: dfx is pinned at `v0.1.14` and scarlettctl at `v0.1.0`, and the local `replace` directives are gone from `go.mod`.

CHANGE: GitHub CI builds and tests every push, and version tags prepare draft source releases from this changelog.

CHANGE: The built-behavior documents moved under `docs/current/`, and the project adopted the practice's changelog and journal conventions.
