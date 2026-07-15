#!/usr/bin/env bash
# Closed-loop concurrency sweep: instead of a fixed request rate, pin N
# concurrent in-flight requests (workers) and let them send as fast as the
# server allows (-rate=0). 

# Prereqs: brew install vegeta; server on :8080 with TrustForwardedFor=true;
# a targets file (see gen_targets.sh).
#
# Usage:
#   ./run_concurrency.sh
#   CONCURRENCY="1 8 64 256 1000" DURATION=10s TARGETS=targets.txt ./run_concurrency.sh
set -euo pipefail

TARGETS="${TARGETS:-targets.txt}"
DURATION="${DURATION:-10s}"
CONCURRENCY="${CONCURRENCY:-1 8 64 256 1000}"   # concurrent in-flight requests

if ! command -v vegeta >/dev/null 2>&1; then
	echo "vegeta not found — install with: brew install vegeta" >&2
	exit 1
fi
if [[ ! -f "$TARGETS" ]]; then
	echo "targets file not found: $TARGETS (generate with gen_targets.sh)" >&2
	exit 1
fi

# Each concurrent worker holds a connection; raise the fd limit so the client
# doesn't run out of sockets before the server runs out of capacity.
ulimit -n 100000 2>/dev/null || echo "note: could not raise ulimit -n" >&2

for c in $CONCURRENCY; do
	echo "=================================================================="
	echo "concurrency=${c}  duration=${DURATION}  targets=${TARGETS}"
	echo "=================================================================="
	# -rate=0: send as fast as possible. -workers/-max-workers pin concurrency.
	vegeta attack \
		-targets="$TARGETS" \
		-rate=0 \
		-workers="$c" \
		-max-workers="$c" \
		-duration="$DURATION" \
		| vegeta report -type=text
	echo
done
