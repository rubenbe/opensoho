#!/bin/sh
# OpenWISP hotplug script, deployed to /etc/hotplug.d/openwisp/opensoho-poe on the target.
# On end-of-cycle, embeds the raw `ubus call poe info` output verbatim into a
# {"type":"OpenSoho","poe":...} object and atomically writes it into the
# openwisp-monitoring upload spool so the send agent POSTs it to the controller.
#
# The file is named with a current UTC timestamp in the agent's own format
# (%d-%m-%Y_%H:%M:%S). The send agent reports the filename as its `time=` query
# param, so the name must be a real, parseable timestamp. Because the agent
# uploads-and-removes files in lexical (≈ within-day chronological) order, a
# "now" name sorts after the in-flight collect files and is therefore evaluated
# when it is the only file left, so the agent stamps the upload with
# `&current=true`. See scripts/dump-radios.sh for the sibling radios dump.
#
# PoE is live telemetry, so a reading is emitted every run (no checksum guard).
#
# Pass -d (or --debug / --stdout) to print the JSON payload to stdout and skip
# the file write, for debugging. This also bypasses the ACTION check.

debug=0
case "$1" in
	-d|--debug|--stdout) debug=1;;
esac

[ "$debug" = 1 ] || [ "$ACTION" = "end-of-cycle" ] || exit 0

OUT_DIR=/tmp/openwisp/monitoring

# Devices without a PoE controller expose no `poe` ubus object; bail quietly.
info=$(ubus call poe info 2>/dev/null) || exit 0
case "$info" in
	'{'*) ;;
	*) exit 0;;
esac

payload="{\"type\":\"OpenSoho\",\"poe\":$info}"

if [ "$debug" = 1 ]; then
	printf '%s\n' "$payload"
	exit 0
fi

mkdir -p "$OUT_DIR"

# Valid-timestamp filename so the send agent's time= param parses and the file
# sorts last among co-present spool files (winning &current=true). Write to a
# dot-prefixed temp (excluded from the agent's "$OUT_DIR"/* glob) then rename
# atomically so the agent never reads a partial file.
ts=$(date -u +'%d-%m-%Y_%H:%M:%S')
out=$OUT_DIR/$ts
tmp=$OUT_DIR/.$ts.tmp
printf '%s' "$payload" > "$tmp" && mv "$tmp" "$out"

# Wake the send instance immediately, the same way the collect instance does.
pid=$(pgrep -f "openwisp-monitoring.*--mode send")
[ -n "$pid" ] && kill -SIGUSR1 "$pid"
