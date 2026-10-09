# CHANGELOG

## Unreleased

FEATURE: `sessionmixer desktop integrate` installs a Linux desktop entry and the sessionmixer mark for the binary that runs it, so sessionmixer launches from the application grid under its own icon; `sessionmixer desktop remove` takes them out again. The window reports `sessionmixer` as its app id, so the desktop pairs a running window with the entry. This replaces the hand-maintained `sessionmixer.desktop` and `sessionmixer.svg` that lived at the repository root.

CHANGE: The module builds from published dependency versions: dfx is pinned at `v0.1.14` and scarlettctl at `v0.1.0`, and the local `replace` directives are gone from `go.mod`.

CHANGE: GitHub CI builds and tests every push, and version tags prepare draft source releases from this changelog.

CHANGE: The built-behavior documents moved under `docs/current/`, and the project adopted the practice's changelog and journal conventions.
