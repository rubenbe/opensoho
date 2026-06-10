#!/bin/sh
# OpenWISP hotplug script.
# Deploy to /etc/hotplug.d/openwisp/opensoho on the target.
# On end-of-cycle, builds a JSON object with a "radios" array (one entry per
# UCI wifi-device, each with name / phy / disabled / iwinfo info / freqlist /
# txpowerlist)
# and atomically writes it to /tmp/openwisp/monitoring/000000_opensoho.json.gz.
# Skips the write when the payload checksum is unchanged.
#
# Pass -d (or --debug / --stdout) to print the JSON payload to stdout and skip
# the file write, for debugging. This also bypasses the ACTION check.

debug=0
case "$1" in
	-d|--debug|--stdout) debug=1;;
esac

[ "$debug" = 1 ] || [ "$ACTION" = "end-of-cycle" ] || exit 0

# newline-separated tokens on stdin -> JSON string array: ["a","b"]
json_str_array() {
	awk 'BEGIN{printf "["} {printf "%s\"%s\"", (NR>1?",":""), $0} END{print "]"}'
}

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
	txpowers=$(ubus call iwinfo txpowerlist "{\"device\":\"$phy\"}")
	disabled=$(uci -q get wireless."$cfg".disabled || echo 0)

	# info: keep only static capability fields
	country=$(echo "$info" | jsonfilter -e '@.country')
	hwmodes=$(echo "$info" | jsonfilter -e '@.hwmodes[*]' | json_str_array)
	htmodes=$(echo "$info" | jsonfilter -e '@.htmodes[*]' | json_str_array)
	info_out="{\"country\":\"$country\",\"hwmodes\":$hwmodes,\"htmodes\":$htmodes}"

	# freqlist: drop runtime "active" (and unused "band")
	n=$(echo "$freqs" | jsonfilter -e '@.results[*].mhz' | wc -l)
	fl=""; fsep=""; i=0
	while [ "$i" -lt "$n" ]; do
		ch=$(echo "$freqs" | jsonfilter -e "@.results[$i].channel")
		mhz=$(echo "$freqs" | jsonfilter -e "@.results[$i].mhz")
		restr=$(echo "$freqs" | jsonfilter -e "@.results[$i].restricted")
		flags=$(echo "$freqs" | jsonfilter -e "@.results[$i].flags[*]" | json_str_array)
		fl="$fl$fsep{\"channel\":$ch,\"mhz\":$mhz,\"restricted\":$restr,\"flags\":$flags}"
		fsep=","; i=$((i + 1))
	done
	freqlist_out="{\"results\":[$fl]}"

	# txpowerlist: drop runtime "active"
	tn=$(echo "$txpowers" | jsonfilter -e '@.results[*].dbm' | wc -l)
	tp=""; tsep=""; i=0
	while [ "$i" -lt "$tn" ]; do
		dbm=$(echo "$txpowers" | jsonfilter -e "@.results[$i].dbm")
		mw=$(echo "$txpowers" | jsonfilter -e "@.results[$i].mw")
		tp="$tp$tsep{\"dbm\":$dbm,\"mw\":$mw}"
		tsep=","; i=$((i + 1))
	done
	txpowerlist_out="{\"results\":[$tp]}"

	payload="$payload$sep{\"name\":\"$cfg\",\"phy\":\"$phy\",\"disabled\":\"$disabled\",\"info\":$info_out,\"freqlist\":$freqlist_out,\"txpowerlist\":$txpowerlist_out}"
	sep=","
done
payload="$payload]}"

if [ "$debug" = 1 ]; then
	printf '%s\n' "$payload"
	exit 0
fi

new=$(printf '%s' "$payload" | md5sum | awk '{print $1}')
old=$(cat "$SUM" 2>/dev/null)

if [ "$new" = "$old" ]; then
	exit 0
fi

tmp=$OUT_DIR/.000000_opensoho.json.gz.tmp
printf '%s' "$payload" | gzip -n > "$tmp" && mv "$tmp" "$OUT" && printf '%s\n' "$new" > "$SUM"
