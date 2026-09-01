// Continuation of dump-radios.uc, include()'d only once dump-radios.uc has
// confirmed the device has at least one UCI wifi-device section - see that
// file's header for why this half is split out. Receives fs / wireless /
// debug / force from the caller's include() scope.

import * as libubus from 'ubus';
import * as libnl from 'nl80211';
import * as digest from 'digest';

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

// nl80211's band enum is declaration order, not frequency order: 60 GHz
// (802.11ad) predates 6 GHz (802.11ax) in the kernel's UAPI header, and
// existing enum values are never renumbered (ABI stability), so 6 GHz was
// appended after 60 GHz - index 3, not 2. Getting this backwards reads the
// (absent, so null) 60 GHz slot instead of the real 6 GHz data - issue #59.
const BAND_IDX = { '2g': 0, '5g': 1, '60g': 2, '6g': 3 };

// One GET_WIPHY dump for all phys, indexed by phy name. split_wiphy_dump is
// required - without it the kernel's reply is truncated to a single wiphy.
// It also means the kernel is free to fragment one phy's reply across
// several messages when it doesn't fit in one (e.g. a tri-band 6 GHz/EHT
// phy) - each fragment carries a different subset of wiphy_bands populated
// and the rest null, so bands must be merged across fragments, not just
// keep the last message (which dropped the 6 GHz band's caps - issue #59).
const nl = libnl.request(libnl.const.NL80211_CMD_GET_WIPHY, libnl.const.NLM_F_DUMP,
	{ split_wiphy_dump: true }) ?? [];
let wiphy = {};
for (let p in nl) {
	if (!p.wiphy_name)
		continue;
	if (!wiphy[p.wiphy_name])
		wiphy[p.wiphy_name] = [];
	let bands = wiphy[p.wiphy_name];
	for (let idx, band in (p.wiphy_bands ?? []))
		if (type(band) == 'object')
			bands[idx] = band;
}

// Raw per-band capability fields for one radio, the input to the htmodes Go
// package. wiphy[phy] is sparse and indexed by nl80211 band enum, so an
// object type() check is needed - a phy without that band has a null there.
function radio_caps(phy, uciband) {
	let b = wiphy[phy]?.[BAND_IDX[uciband]];
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

// exit() segfaults ucode when called from an include()'d file (unrelated to
// which modules it imported), so this falls through instead of returning
// early - dump-radios.uc's exit()s are all fine, since they run before the
// include() below is ever reached.
if (debug) {
	print(payload + '\n');
} else {
	let new_sum = digest.md5(sig);
	let old_sum = trim(fs.readfile(SUM) ?? '');

	if (force || new_sum != old_sum) {
		let tmp = OUT_DIR + '/.000000_opensoho.json.gz.tmp';
		let gz = fs.popen(`gzip -n -c > ${tmp}`, 'w');
		gz.write(payload);
		gz.close();
		fs.rename(tmp, OUT);
		fs.writefile(SUM, new_sum + '\n');
	}
}
