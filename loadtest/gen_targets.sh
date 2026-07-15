#!/usr/bin/env bash
# Generate a vegeta targets file on stdout.
#
# A vegeta target is a request line followed by optional header lines, with a
# blank line between targets:
#
#   GET http://localhost:8080/
#   X-Forwarded-For: 10.0.0.1
#
#   GET http://localhost:8080/
#   X-Forwarded-For: 10.0.0.2
#
# vegeta round-robins through the targets it's given.
#
# Usage:
#   ./gen_targets.sh > targets.txt              # 1000 rotating client IPs
#   MODE=one ./gen_targets.sh > targets_one.txt # single hot client
#   CLIENTS=5000 URL=http://localhost:8080/ ./gen_targets.sh > targets.txt
#
# The server must run with TrustForwardedFor=true for the X-Forwarded-For values
# to actually be used as the client key.
set -euo pipefail

URL="${URL:-http://localhost:8080/}"
CLIENTS="${CLIENTS:-1000}"   # number of distinct fake client IPs (MODE=many)
MODE="${MODE:-many}"         # many = rotate XFF across CLIENTS IPs; one = single IP

emit_target() {
	# $1 = X-Forwarded-For value
	printf 'GET %s\n' "$URL"
	printf 'X-Forwarded-For: %s\n' "$1"
	printf '\n'
}

case "$MODE" in
	many)
		for ((i = 1; i <= CLIENTS; i++)); do
			# Spread across 10.x.x.x. Good for up to ~16M distinct clients.
			o2=$(((i / 65536) % 256))
			o3=$(((i / 256) % 256))
			o4=$((i % 256))
			emit_target "10.${o2}.${o3}.${o4}"
		done
		;;
	one)
		emit_target "10.0.0.1"
		;;
	*)
		echo "unknown MODE: $MODE (want 'many' or 'one')" >&2
		exit 1
		;;
esac
