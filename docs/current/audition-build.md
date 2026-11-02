# audition builds

`make audition` builds the current sessionmixer working tree for the Ubuntu 24.04 amd64 studio machines. it includes staged, unstaged, and untracked source files without requiring a commit. the build runs inside a Podman container so CGO links against Ubuntu 24.04's native libraries even when the development host runs a newer distribution.

## build

the host needs linux amd64, rootless Podman, git, make, bash, and tar. the first run needs network access to fetch the container images, ubuntu packages, and Go modules. the container supplies Go and the native build dependencies.

from the repository root:

```bash
make audition
```

the result is a timestamped archive in the ignored `dist/` directory:

```text
dist/sessionmixer-audition-<commit>[-dirty]-<timestamp>-ubuntu24.04-amd64.tar.gz
```

keep source edits paused during the build: the source directory is mounted read-only into the container, but host edits still change that directory. the build uses `go.mod` and `go.sum` with `GOWORK=off` and `-mod=readonly`; local `go.work` overrides and sibling dependency checkouts do not participate. dependencies must be fetchable as modules; filesystem replacements outside this checkout are not mounted.

the archive contains a directory of the same name, holding:

- `sessionmixer`: the executable.
- `build-info.txt`: full commit, clean/dirty state, UTC timestamp, target, Go version, and builder image ID.
- `source-status.txt`: the checkout's git status at build start, including untracked files and excluding `dist/`. this is identification, not a copy of the source changes.
- `help.txt`: output from loading the executable and running `sessionmixer --help` inside Ubuntu 24.04. this check does not access hardware.
- `linked-libraries.txt`: the executable's resolved shared libraries in the builder.
- `SHA256SUMS`: the executable's checksum.

the target builds `./cmd/sessionmixer` and performs the executable-load check. it does not run the test suite, install locally, or deploy to a studio machine. use `make test` for the repository verification gate. sessionmixer has no version command or embedded audition stamp; source identification lives in the archive name and `build-info.txt`.

## run on the studio machine

copy the archive to the Ubuntu 24.04 machine and extract it into a convenient directory:

```bash
tar -xzf <archive>.tar.gz
cd <archive-directory>
sha256sum -c SHA256SUMS
./sessionmixer run
```

use the explicit `./sessionmixer` path to select this audition build. it uses your ordinary `~/.config/sessionmixer/session.yaml` and `ui.json`. configuration files are not packaged. to add a desktop entry for this binary, run `./sessionmixer desktop integrate` from a directory you intend to keep; the entry records the binary's absolute path.

this is a dynamically linked executable. the studio machine needs the usual GTK, ALSA, OpenGL, and X11 runtime libraries. on Ubuntu 24.04, the runtime packages are:

```bash
sudo apt-get install libgtk-3-0t64 libasound2t64 libgl1 libstdc++6 \
    libxcursor1 libxi6 libxinerama1 libxrandr2
```

the container load check confirms executable/library compatibility. GUI rendering, hardware controls, metering, and resize performance still need a hands-on check on the studio machine.

## compare a baseline

the current builder can build another sessionmixer checkout, including a revision that predates the audition target:

```bash
git worktree add --detach /tmp/sessionmixer-baseline <revision>
make audition AUDITION_SOURCE=/tmp/sessionmixer-baseline
```

both archives land in the invoking checkout's `dist/`. a clean baseline has no `-dirty` suffix. extract baseline and candidate into separate directories, and launch their respective executables to compare behavior. remove the temporary worktree with `git worktree remove /tmp/sessionmixer-baseline` when finished.

## builder and caches

the Makefile delegates to `tools/audition-build.sh` and `tools/audition.Containerfile`, adapted from baab's audition builder. the image uses Ubuntu 24.04 and the Go patch version declared by the source checkout's `go.mod`. `GOAMD64=v1` avoids assuming a newer studio CPU. no host Go build flags or native libraries are imported.

Podman retains the builder image. module downloads and compilation caches live under `${XDG_CACHE_HOME:-$HOME/.cache}/sessionmixer-audition`; set `AUDITION_CACHE_DIR` to use another directory. compilation caches are separated by builder image ID so a changed native toolchain cannot reuse old CGO objects. repeated builds reuse these caches. `make clean` removes audition archives under `dist/` along with the ordinary project binaries; it leaves the builder image and caches intact.
