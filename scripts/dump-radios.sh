#!/bin/sh
# OpenWISP hotplug script.
# Deploy to /etc/hotplug.d/openwisp/opensoho on the target.
# On end-of-cycle, builds a JSON object with a "radios" array (one entry per
# UCI wifi-device, each with name / phy / disabled / iwinfo info / freqlist)
# and atomically writes it to /tmp/openwisp/monitoring/000000_opensoho.json.gz.
# Skips the write when the payload is unchanged and the target file exists.

[ "$ACTION" = "end-of-cycle" ] || exit 0

STATE_DIR=/tmp/opensoho
OUT_DIR=/tmp/openwisp/monitoring
OUT=$OUT_DIR/000000_opensoho.json.gz
SUM=$STATE_DIR/dump-radios.md5

mkdir -p "$STATE_DIR" "$OUT_DIR"

payload='{"type":"OpenSoho","radios":['
sep=""
for cfg in $(uci -q show wireless | sed -n 's/^wireless\.\(radio[0-9]*\)=wifi-device$/\1/p'); do
	cpath=$(uci -q get wireless."$cfg".path)
	phy=""
	for p in /sys/class/ieee80211/phy*; do
		[ -e "$p" ] || continue
		rp=$(readlink -f "$p/device" 2>/dev/null)
		case "$rp" in */$cpath) phy=${p##*/}; break;; esac
	done
	[ -n "$phy" ] || continue
	info=$(ubus call iwinfo info "{\"device\":\"$phy\"}")
	freqs=$(ubus call iwinfo freqlist "{\"device\":\"$phy\"}")
	disabled=$(uci -q get wireless."$cfg".disabled || echo 0)
	payload="$payload$sep{\"name\":\"$cfg\",\"phy\":\"$phy\",\"disabled\":$disabled,\"info\":$info,\"freqlist\":$freqs}"
	sep=","
done
payload="$payload]}"

new=$(printf '%s' "$payload" | md5sum | awk '{print $1}')
old=$(cat "$SUM" 2>/dev/null)

if [ "$new" = "$old" ] && [ -f "$OUT" ]; then
	exit 0
fi

tmp=$OUT_DIR/.000000_opensoho.json.gz.tmp
printf '%s' "$payload" | gzip -n > "$tmp" && mv "$tmp" "$OUT" && printf '%s\n' "$new" > "$SUM"
