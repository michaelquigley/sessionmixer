#!/usr/bin/env bash
# builds the current source tree, including uncommitted files, for ubuntu 24.04.
# an optional source directory permits baseline builds from a separate checkout.
set -euo pipefail

fail() { printf "audition: %s\n" "$*" >&2; exit 1; }

[[ $# -le 1 ]] || fail "usage: bash tools/audition-build.sh [source-directory]"
for tool in podman git tar; do
    command -v "$tool" >/dev/null || fail "required tool '$tool' was not found"
done
[[ $(uname -s) == Linux && $(uname -m) == x86_64 ]] || fail "this builder requires a linux amd64 host"

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
source_dir=$(cd -- "${1:-.}" && pwd)
grep -qx 'module github.com/michaelquigley/sessionmixer' "$source_dir/go.mod" || fail "source '$source_dir' is not a sessionmixer checkout"
[[ $(git -C "$source_dir" rev-parse --show-toplevel) == "$source_dir" ]] || fail "source '$source_dir' must be the checkout root"

go_version=$(awk '$1 == "go" { print $2; exit }' "$source_dir/go.mod")
[[ $go_version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "expected a Go patch version in '$source_dir/go.mod'"
commit=$(git -C "$source_dir" rev-parse HEAD)
status=$(git -C "$source_dir" status --porcelain --untracked-files=all -- . ':!dist')
state=clean
[[ -z $status ]] || state=dirty
version="audition-${commit:0:12}"
[[ $state == clean ]] || version+="-dirty"
built_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)
stamp=$(date -u +%Y%m%dT%H%M%SZ)
bundle="sessionmixer-${version}-${stamp}-ubuntu24.04-amd64"

# outputs belong to the invoking sessionmixer checkout, even when building a baseline.
out_dir=$(cd -- "$script_dir/.." && pwd)/dist
cache_dir=${AUDITION_CACHE_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/sessionmixer-audition}
mkdir -p "$out_dir" "$cache_dir/modules"
cache_dir=$(cd -- "$cache_dir" && pwd)
stage_dir=$(mktemp -d "$out_dir/.audition-XXXXXXXX")
trap 'rm -rf -- "$stage_dir"' EXIT
mkdir "$stage_dir/$bundle"

image="localhost/sessionmixer-audition-builder:24.04-go${go_version}"
printf "==> preparing builder '%s'\n" "$image"
podman build --build-arg "GO_VERSION=$go_version" \
    --iidfile "$stage_dir/image-id" -t "$image" \
    -f "$script_dir/audition.Containerfile" "$script_dir"
image_id=$(cat "$stage_dir/image-id")
# a changed native toolchain must not reuse CGO objects from an older image.
build_cache="$cache_dir/build/${image_id#sha256:}"
mkdir -p "$build_cache"

printf "==> building source '%s' (%s)\n" "$source_dir" "$state"
podman run --rm --userns=keep-id \
    -v "$source_dir:/src:ro" \
    -v "$cache_dir/modules:/cache/modules" \
    -v "$build_cache:/cache/build" \
    -v "$stage_dir/$bundle:/out" \
    -w /src "$image_id" bash -euo pipefail -c '
        # metadata comes from the host so separate git worktrees work as well.
        go build -mod=readonly -trimpath -buildvcs=false -ldflags "-s -w" -o /out/sessionmixer ./cmd/sessionmixer
        /out/sessionmixer --help > /out/help.txt
        ldd /out/sessionmixer > /out/linked-libraries.txt
        cd /out
        sha256sum sessionmixer > SHA256SUMS
    '

cat > "$stage_dir/$bundle/build-info.txt" <<EOF
version: $version
commit: $commit
source state: $state
built at: $built_at
target: ubuntu 24.04, linux/amd64, GOAMD64=v1
go: $go_version
cgo: enabled
builder image: $image_id
verification: executable loaded and help command passed inside ubuntu 24.04
tests: not run by audition; use make test separately
EOF
printf '%s\n' "$status" > "$stage_dir/$bundle/source-status.txt"
tar -czf "$stage_dir/$bundle.tar.gz" -C "$stage_dir" "$bundle"
mv -- "$stage_dir/$bundle.tar.gz" "$out_dir/$bundle.tar.gz"
printf "==> archive '%s'\n" "$out_dir/$bundle.tar.gz"
printf "==> build and executable load check passed; tests were not run (use 'make test')\n"
