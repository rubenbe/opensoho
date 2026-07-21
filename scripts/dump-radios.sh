#!/bin/sh
# OpenWISP hotplug script, deployed to /etc/hotplug.d/openwisp/opensoho on the target.
# On end-of-cycle, builds a JSON object with a "radios" array (one entry per
# UCI wifi-device, each with name / phy / disabled and the raw iwinfo info /
# freqlist / txpowerlist) and atomically writes it to
# /tmp/openwisp/monitoring/000000_opensoho.json.gz.
#
# The raw ubus outputs are embedded verbatim (the server ignores fields it
# doesn't know). To avoid rewriting the file when only runtime values change,
# the checksum is computed over a "signature" built from the stable capability
# fields only, not over the full payload.
#
# Pass -d (or --debug / --stdout) to print the JSON payload to stdout and skip
# the file write, for debugging. This also bypasses the ACTION check.

debug=0
case "$1" in
	-d|--debug|--stdout) debug=1;;
esac

[ "$debug" = 1 ] || [ "$ACTION" = "end-of-cycle" ] || exit 0

STATE_DIR=/tmp/opensoho
OUT_DIR=/tmp/openwisp/monitoring
OUT=$OUT_DIR/000000_opensoho.json.gz
SUM=$STATE_DIR/dump-radios.md5

mkdir -p "$STATE_DIR" "$OUT_DIR"

payload='{"type":"OpenSoho","radios":['
sig=""
sep=""
for cfg in $(uci -q show wireless | sed -n 's/^wireless\.\(radio[0-9]*\)=wifi-device$/\1/p'); do
	cpath=$(uci -q get wireless."$cfg".path)
	# Combo chips expose several bands as separate phys sharing one device
	# path; OpenWrt disambiguates them with a "<path>+N" suffix. readlink strips
	# the +N (all such phys resolve to the same base path), so match the base
	# path and pick the Nth phy (0-based) among those that share it.
	base=${cpath%+*}
	offset=${cpath##*+}
	case "$offset" in ""|*[!0-9]*) offset=0;; esac
	phy=""
	n=0
	for p in $(ls -d /sys/class/ieee80211/phy* 2>/dev/null | sort -V); do
		[ -e "$p" ] || continue
		rp=$(readlink -f "$p/device" 2>/dev/null)
		case "$rp" in
			*/$base)
				[ "$n" = "$offset" ] && { phy=${p##*/}; break; }
				n=$((n+1));;
		esac
	done
	[ -n "$phy" ] || continue
	info=$(ubus call iwinfo info "{\"device\":\"$phy\"}")
	freqs=$(ubus call iwinfo freqlist "{\"device\":\"$phy\"}")
	txpowers=$(ubus call iwinfo txpowerlist "{\"device\":\"$phy\"}")
	disabled=$(uci -q get wireless."$cfg".disabled || echo 0)

	# Embed the raw ubus outputs verbatim; the server ignores unknown fields.
	payload="$payload$sep{\"name\":\"$cfg\",\"phy\":\"$phy\",\"disabled\":\"$disabled\",\"info\":$info,\"freqlist\":$freqs,\"txpowerlist\":$txpowers}"
	sep=","

	# Signature: only the stable capability fields, so runtime values
	# (info.channel/txpower, results[].active, ...) don't trigger a rewrite.
	sig="$sig|$cfg|$disabled"
	sig="$sig|$(echo "$info" | jsonfilter -e '@.country' -e '@.hwmodes[*]' -e '@.htmodes[*]')"
	sig="$sig|$(echo "$freqs" | jsonfilter -e '@.results[*].channel' -e '@.results[*].mhz' -e '@.results[*].restricted' -e '@.results[*].flags[*]')"
	sig="$sig|$(echo "$txpowers" | jsonfilter -e '@.results[*].dbm' -e '@.results[*].mw')"
done
payload="$payload]}"

if [ "$debug" = 1 ]; then
	printf '%s\n' "$payload"
	exit 0
fi

new=$(printf '%s' "$sig" | md5sum | awk '{print $1}')
old=$(cat "$SUM" 2>/dev/null)

if [ "$new" = "$old" ]; then
	exit 0
fi

tmp=$OUT_DIR/.000000_opensoho.json.gz.tmp
printf '%s' "$payload" | gzip -n > "$tmp" && mv "$tmp" "$OUT" && printf '%s\n' "$new" > "$SUM"
