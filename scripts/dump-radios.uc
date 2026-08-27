#!/usr/bin/ucode
// OpenWISP hotplug script, deployed to /etc/hotplug.d/openwisp/opensoho on
// the target. On end-of-cycle, builds a JSON object with a "radios" array
// (one entry per UCI wifi-device, each with name / phy / ifname / band /
// radio_index / disabled and the raw iwinfo info / freqlist / txpowerlist)
// and atomically writes it to /tmp/openwisp/monitoring/000000_opensoho.json.gz.
//
// "radio_index" mirrors UCI's "radio" option, present only when the wiphy
// advertises multiple radios. "ifname" is the real interface iwinfo was
// queried against - see radio_ifname()/radio_capability_from_board() below.
//
// The raw ubus outputs are embedded verbatim (the server ignores fields it
// doesn't know). To avoid rewriting the file when only runtime values change,
// the checksum is computed over a "signature" built from the stable capability
// fields only, not over the full payload.
//
// Pass -d (or --debug / --stdout) to print the JSON payload to stdout and skip
// the file write, for debugging. This also bypasses the ACTION check.
//
// Pass -f (or --force) to skip the checksum comparison and always write/upload
// the file, even if the signature is unchanged since the last run.

import * as fs from 'fs';
import * as ucilib from 'uci';
import * as libubus from 'ubus';
import * as digest from 'digest';

let debug = false, force = false;
for (let arg in ARGV) {
	switch (arg) {
		case '-d': case '--debug': case '--stdout': debug = true; break;
		case '-f': case '--force': force = true; break;
	}
}

if (!debug && !force && getenv('ACTION') != 'end-of-cycle') {
	warn('use -f to force re-upload\n');
	exit(0);
}

const STATE_DIR = '/tmp/opensoho';
const OUT_DIR = '/tmp/openwisp/monitoring';
const OUT = OUT_DIR + '/000000_opensoho.json.gz';
const SUM = STATE_DIR + '/dump-radios.md5';

fs.mkdir(STATE_DIR);
fs.mkdir(OUT_DIR);

const ubus = libubus.connect();

// Real per-radio interface names, so iwinfo can be queried per radio instead
// of per (possibly shared) phy.
const wireless_status = ubus.call('network.wireless', 'status') ?? {};

// ifname of the first up interface on a UCI wifi-device section, or null if
// the radio is disabled/unconfigured.
function radio_ifname(cfg) {
	return wireless_status[cfg]?.interfaces?.[0]?.ifname;
}

// Fallback for a disabled/unconfigured radio (no ifname): backfills htmodes
// from /etc/board.json, the same per-radio data mac80211.uc used to write
// this radio's "option band"/"option htmode" into UCI in the first place.
// Static file, no interface needed.
const board_raw = fs.readfile('/etc/board.json');
const board = board_raw ? json(board_raw) : null;

function radio_capability_from_board(phy, ridx) {
	// Only meaningful on confirmed single-wiphy multi-radio hardware.
	if (ridx === null || ridx === '' || !board)
		return null;

	let phy_info = board?.wlan?.[phy]?.info;
	if (!phy_info)
		return null;

	let radio = filter(phy_info.radios ?? [], r => r.index == ridx)[0];
	if (!radio)
		return null;

	let band_name = null;
	for (let b in ['2G', '5G', '6G', '60G']) {
		if (radio.bands?.[b] != null) {
			band_name = b;
			break;
		}
	}
	if (!band_name)
		return null;

	let band = phy_info.bands?.[band_name];
	if (!band)
		return null;

	// board.json already lists the exact supported modes per band; "NOHT"
	// isn't a real htmode so it's dropped.
	let modes = filter(band.modes ?? [], m => m != 'NOHT');
	if (!length(modes))
		return null;

	return { htmodes: modes };
}

// Cut the freqlist down to one band. Only applied when radio_index is set
// (confirmed single-wiphy multi-radio, e.g. MT7996) - a plain switchable
// single radio (band pinned, no "radio" option) keeps its full freqlist.
function scope_freqlist(freqs, uciband, ridx) {
	if (ridx === null || ridx === '')
		return freqs;
	if (!(uciband in { '2g': 1, '5g': 1, '6g': 1, '60g': 1 }))
		return freqs;
	let ghz = int(substr(uciband, 0, length(uciband) - 1));
	return { results: filter(freqs?.results ?? [], f => f.band == ghz) };
}

let radios = [];
let sig = '';

let wireless = ucilib.cursor().get_all('wireless') ?? {};
for (let cfg, s in wireless) {
	if (s['.type'] != 'wifi-device')
		continue;

	let band = s.band ?? '';
	let ridx = s.radio ?? '';

	// Resolve the phy via iwinfo's own resolver.
	let phy = ubus.call('iwinfo', 'phyname', { section: cfg })?.phyname;
	if (!phy)
		continue;

	// Prefer the real per-radio interface over the phy; falls back to the
	// phy when the radio has no interface up yet.
	let ifname = radio_ifname(cfg);
	let iwinfo_dev = ifname ?? phy;

	let info;
	if (ifname) {
		info = ubus.call('iwinfo', 'info', { device: ifname }) ?? {};
	} else {
		info = radio_capability_from_board(phy, ridx) ?? ubus.call('iwinfo', 'info', { device: phy }) ?? {};
	}
	let freqs = scope_freqlist(ubus.call('iwinfo', 'freqlist', { device: iwinfo_dev }), band, ridx);
	let txpowers = ubus.call('iwinfo', 'txpowerlist', { device: iwinfo_dev }) ?? {};
	let disabled = s.disabled ?? '0';

	push(radios, {
		name: cfg,
		phy: phy,
		ifname: ifname ?? '',
		band: band,
		radio_index: ridx,
		disabled: disabled,
		info: info,
		freqlist: freqs,
		txpowerlist: txpowers,
	});

	// Signature: only the stable capability fields, so runtime values
	// (info.channel/txpower, results[].active, ...) don't trigger a rewrite.
	sig += sprintf('|%s|%s|%s|%s', cfg, band, ridx, disabled);
	sig += sprintf('|%s|%J|%J', info.country ?? '', info.hwmodes ?? [], info.htmodes ?? []);
	for (let f in freqs.results ?? [])
		sig += sprintf('|%d|%d|%d|%J', f.channel, f.mhz, f.restricted ? 1 : 0, f.flags ?? []);
	for (let p in txpowers.results ?? [])
		sig += sprintf('|%d|%d', p.dbm, p.mw);
}

ubus.disconnect();

let payload = sprintf('%J', { type: 'OpenSoho', radios: radios });

if (debug) {
	print(payload + '\n');
	exit(0);
}

let new_sum = digest.md5(sig);
let old_sum = trim(fs.readfile(SUM) ?? '');

if (!force && new_sum == old_sum)
	exit(0);

let tmp = OUT_DIR + '/.000000_opensoho.json.gz.tmp';
let gz = fs.popen(`gzip -n -c > ${tmp}`, 'w');
gz.write(payload);
gz.close();
fs.rename(tmp, OUT);
fs.writefile(SUM, new_sum + '\n');
