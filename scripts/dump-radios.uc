#!/usr/bin/ucode
// OpenWISP hotplug script, deployed to /etc/hotplug.d/openwisp/opensoho on
// the target. On end-of-cycle, builds a JSON object with a "radios" array
// (one entry per UCI wifi-device, each with name / phy / ifname / band /
// radio_index / disabled and the raw iwinfo info / freqlist / txpowerlist /
// caps) and atomically writes it to
// /tmp/openwisp/monitoring/000000_opensoho.json.gz.
//
// "radio_index" mirrors UCI's "radio" option, present only when the wiphy
// advertises multiple radios. "ifname" is the real interface iwinfo was
// queried against - see radio_ifname() below. "caps" is the raw per-band
// nl80211 capability fields - see radio_caps() below - decoded server-side by
// the htmodes Go package, since iwinfo's own htmode list is a whole-wiphy
// union and wrong on a shared wiphy.
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
import * as libnl from 'nl80211';
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

const BAND_IDX = { '2g': 0, '5g': 1, '6g': 2, '60g': 3 };

// One GET_WIPHY dump for all phys, indexed by phy name. split_wiphy_dump is
// required - without it the kernel's reply is truncated to a single wiphy.
const nl = libnl.request(libnl.const.NL80211_CMD_GET_WIPHY, libnl.const.NLM_F_DUMP,
	{ split_wiphy_dump: true }) ?? [];
let wiphy = {};
for (let p in nl)
	if (p.wiphy_name && p.wiphy_bands)
		wiphy[p.wiphy_name] = p;

// Raw per-band capability fields for one radio, the input to the htmodes Go
// package. wiphy_bands is sparse and indexed by nl80211 band enum, so an
// object type() check is needed - a phy without that band has a null there.
function radio_caps(phy, uciband) {
	let b = wiphy[phy]?.wiphy_bands?.[BAND_IDX[uciband]];
	if (type(b) != 'object')
		return null;

	let caps = {};
	if (b.ht_capa != null)
		caps.ht_capa = b.ht_capa;
	if (b.vht_capa != null)
		caps.vht_capa = b.vht_capa;

	// AP is the iftype opensoho configures; fall back to the first entry.
	let d = filter(b.iftype_data ?? [], e => e.iftypes?.ap)[0] ?? b.iftype_data?.[0];
	if (d) {
		if (d.he_cap_phy)
			caps.he_cap_phy = d.he_cap_phy;
		if (d.eht_cap_phy)
			caps.eht_cap_phy = d.eht_cap_phy;
	}
	return length(caps) ? caps : null;
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

	let info = ubus.call('iwinfo', 'info', { device: iwinfo_dev }) ?? {};
	let caps = radio_caps(phy, band);
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
		caps: caps,
		freqlist: freqs,
		txpowerlist: txpowers,
	});

	// Signature: only the stable capability fields, so runtime values
	// (info.channel/txpower, results[].active, ...) don't trigger a rewrite.
	sig += sprintf('|%s|%s|%s|%s', cfg, band, ridx, disabled);
	sig += sprintf('|%s|%J|%J|%J', info.country ?? '', info.hwmodes ?? [], info.htmodes ?? [], caps);
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
