#!/bin/sh
# OpenWISP hotplug script, deployed to /etc/hotplug.d/openwisp/opensoho on the target.
# On end-of-cycle, builds a JSON object with a "radios" array (one entry per
# UCI wifi-device, each with name / phy / ifname / band / radio_index /
# disabled and the raw iwinfo info / freqlist / txpowerlist) and atomically
# writes it to /tmp/openwisp/monitoring/000000_opensoho.json.gz.
#
# "radio_index" mirrors UCI's "radio" option, present only when the wiphy
# advertises multiple radios. "ifname" is the real interface iwinfo was
# queried against - see radio_ifname()/radio_capability_from_board() below.
#
# The raw ubus outputs are embedded verbatim (the server ignores fields it
# doesn't know). To avoid rewriting the file when only runtime values change,
# the checksum is computed over a "signature" built from the stable capability
# fields only, not over the full payload.
#
# Pass -d (or --debug / --stdout) to print the JSON payload to stdout and skip
# the file write, for debugging. This also bypasses the ACTION check.
#
# Pass -f (or --force) to skip the checksum comparison and always write/upload
# the file, even if the signature is unchanged since the last run.

debug=0
force=0
for arg in "$@"; do
	case "$arg" in
		-d|--debug|--stdout) debug=1;;
		-f|--force) force=1;;
	esac
done

if [ "$debug" = 1 ] || [ "$force" = 1 ] || [ "$ACTION" = "end-of-cycle" ]; then :
else
	echo "use -f to force re-upload" >&2
	exit 0
fi

STATE_DIR=/tmp/opensoho
OUT_DIR=/tmp/openwisp/monitoring
OUT=$OUT_DIR/000000_opensoho.json.gz
SUM=$STATE_DIR/dump-radios.md5

mkdir -p "$STATE_DIR" "$OUT_DIR"

# Real per-radio interface names, so iwinfo can be queried per radio instead
# of per (possibly shared) phy. Fetched once and cached.
wireless_status=$(ubus call network.wireless status 2>/dev/null)

# ifname of the first up interface on a UCI wifi-device section, or "" if the
# radio is disabled/unconfigured.
radio_ifname() {
	printf '%s' "$wireless_status" | jsonfilter -e "@.$1.interfaces[0].ifname" 2>/dev/null
}

# Fallback to static boards.json
board_json=$(cat /etc/board.json 2>/dev/null)

radio_capability_from_board() {
	phy=$1
	ridx=$2
	# Only meaningful on confirmed single-wiphy multi-radio hardware.
	[ -n "$ridx" ] || return 1
	[ -n "$board_json" ] || return 1

	phy_info=$(printf '%s' "$board_json" | jsonfilter -e "@.wlan[\"$phy\"].info" 2>/dev/null)
	[ -n "$phy_info" ] || return 1

	radio=$(printf '%s' "$phy_info" | jsonfilter -e "@.radios[@.index=$ridx]" 2>/dev/null)
	[ -n "$radio" ] || return 1

	band_name=""
	for b in 2G 5G 6G 60G; do
		v=$(printf '%s' "$radio" | jsonfilter -e "@.bands[\"$b\"]" 2>/dev/null)
		case "$v" in ""|false) ;; *) band_name=$b; break;; esac
	done
	[ -n "$band_name" ] || return 1

	band=$(printf '%s' "$phy_info" | jsonfilter -e "@.bands[\"$band_name\"]" 2>/dev/null)
	[ -n "$band" ] || return 1

	# board.json already lists the exact supported modes per band.
	# "NOHT" isn't a real htmode so it's dropped.
	modes=$(printf '%s' "$band" | jsonfilter -e '@.modes[*]' 2>/dev/null | \
		grep -v '^NOHT$' | sed 's/.*/"&"/' | tr '\n' ',' | sed 's/,$//')
	[ -n "$modes" ] || return 1

	printf '{"htmodes":[%s]}' "$modes"
}

# Cut the freqlist down to one band. Only applied when radio_index is set
# (confirmed single-wiphy multi-radio, e.g. MT7996) - a plain switchable
# single radio (band pinned, no "radio" option) keeps its full freqlist.
scope_freqlist() {
	case "$3" in
		"") printf '%s' "$1"; return;;
	esac
	case "$2" in
		2g|5g|6g|60g) ;;
		*) printf '%s' "$1"; return;;
	esac
	ghz=${2%g}
	rows=$(printf '%s' "$1" | jsonfilter -e "@.results[@.band=$ghz]" | tr '\n' ',' | sed 's/,$//')
	printf '{"results":[%s]}' "$rows"
}

payload='{"type":"OpenSoho","radios":['
sig=""
sep=""
for cfg in $(uci -q show wireless | sed -n 's/^wireless\.\(radio[0-9]*\)=wifi-device$/\1/p'); do
	# Bands are required by single-phy multi-radio e.g. MT7996.
	band=$(uci -q get wireless."$cfg".band)
	ridx=$(uci -q get wireless."$cfg".radio)

	# Resolve the phy via iwinfo's own resolver.
	phy=$(ubus call iwinfo phyname "{\"section\":\"$cfg\"}" 2>/dev/null | jsonfilter -e '@.phyname' 2>/dev/null)
	[ -n "$phy" ] || continue

	# Prefer the real per-radio interface over the phy; falls back to the
	# phy when the radio has no interface up yet.
	ifname=$(radio_ifname "$cfg")
	iwinfo_dev=${ifname:-$phy}

	if [ -n "$ifname" ]; then
		info=$(ubus call iwinfo info "{\"device\":\"$ifname\"}")
	else
		info=$(radio_capability_from_board "$phy" "$ridx")
		[ -n "$info" ] || info=$(ubus call iwinfo info "{\"device\":\"$phy\"}")
	fi
	freqs=$(scope_freqlist "$(ubus call iwinfo freqlist "{\"device\":\"$iwinfo_dev\"}")" "$band" "$ridx")
	txpowers=$(ubus call iwinfo txpowerlist "{\"device\":\"$iwinfo_dev\"}")
	disabled=$(uci -q get wireless."$cfg".disabled || echo 0)

	# Embed the raw ubus outputs verbatim; the server ignores unknown fields.
	payload="$payload$sep{\"name\":\"$cfg\",\"phy\":\"$phy\",\"ifname\":\"$ifname\",\"band\":\"$band\",\"radio_index\":\"$ridx\",\"disabled\":\"$disabled\",\"info\":$info,\"freqlist\":$freqs,\"txpowerlist\":$txpowers}"
	sep=","

	# Signature: only the stable capability fields, so runtime values
	# (info.channel/txpower, results[].active, ...) don't trigger a rewrite.
	sig="$sig|$cfg|$band|$ridx|$disabled"
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

if [ "$force" != 1 ] && [ "$new" = "$old" ]; then
	exit 0
fi

tmp=$OUT_DIR/.000000_opensoho.json.gz.tmp
printf '%s' "$payload" | gzip -n > "$tmp" && mv "$tmp" "$OUT" && printf '%s\n' "$new" > "$SUM"
