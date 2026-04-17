#!/usr/bin/env bash
# Fetch chain-command-blocker binaries from the source repo's release and
# place them under bin/ so they can be committed to this plugin repo.
#
# The plugin repo is distributed as-is to users, so bin/ must stay in sync
# with a specific source tag. The plugin version itself is managed
# independently by tagpr on the next release cut.
#
# Usage: ./scripts/sync-from-src.sh <src-tag>
#   e.g. ./scripts/sync-from-src.sh v1.0.0

set -euo pipefail

SRC_REPO=gurisugi/chain-command-blocker-src

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <src-tag>" >&2
  echo "Example: $0 v1.0.0" >&2
  exit 2
fi

TAG="$1"

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN_DIR="${REPO_ROOT}/bin"

tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

echo "Downloading ${SRC_REPO}@${TAG} release assets..."
gh release download "$TAG" \
  --repo "$SRC_REPO" \
  --dir "$tmp_dir" \
  --pattern 'chain-command-blocker_*.tar.gz' \
  --pattern 'checksums.txt'

echo "Verifying checksums..."
(cd "$tmp_dir" && shasum -a 256 -c checksums.txt --ignore-missing)

echo "Extracting binaries to ${BIN_DIR}..."
mkdir -p "$BIN_DIR"
for archive in "$tmp_dir"/chain-command-blocker_*.tar.gz; do
  name=$(basename "$archive")
  name=${name#chain-command-blocker_}
  name=${name%.tar.gz}
  work=$(mktemp -d)
  tar -xzf "$archive" -C "$work"
  install -m 0755 "$work/chain-command-blocker" "${BIN_DIR}/chain-command-blocker-${name}"
  rm -rf "$work"
done

echo "Done. Binaries synced from ${SRC_REPO}@${TAG}:"
ls -la "${BIN_DIR}"
echo
echo "Next steps:"
echo "  git add bin/"
echo "  git commit -m 'Sync binaries from chain-command-blocker-src@${TAG}'"
echo "  git push"
