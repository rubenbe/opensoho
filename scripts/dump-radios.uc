#!/usr/bin/ucode
// OpenWISP hotplug script, deployed to /etc/hotplug.d/openwisp/opensoho on
// the target. On end-of-cycle, builds a JSON object with a "radios" array
// (one entry per UCI wifi-device, each with name / phy / ifname / band /
// radio_index / disabled and the raw iwinfo info / freqlist / txpowerlist /
// caps) and atomically writes it to
// /tmp/openwisp/monitoring/000000_opensoho.json.gz.
//
// This file only imports modules ucode itself always ships (fs/uci). The
// per-radio work needs nl80211 and digest, which come from wifi-scripts and
// aren't installed on wifi-less hardware (switches, plain routers) - ucode
// resolves `import` at compile time, so referencing them here at all would
// hard-fail on that hardware even if the code path is never reached at
// runtime. dump-radios-caps.uc (deployed to
// /usr/share/opensoho/dump-radios-caps.uc) carries that half; it's
// include()'d - and so only compiled - once a UCI wifi-device section is
// confirmed to exist.
//
// Pass -d (or --debug / --stdout) to print the JSON payload to stdout and skip
// the file write, for debugging. This also bypasses the ACTION check.
//
// Pass -f (or --force) to skip the checksum comparison and always write/upload
// the file, even if the signature is unchanged since the last run.

import * as fs from 'fs';
import * as ucilib from 'uci';

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

let wireless = ucilib.cursor().get_all('wireless') ?? {};

let has_radios = false;
for (let cfg, s in wireless)
	if (s['.type'] == 'wifi-device') {
		has_radios = true;
		break;
	}

if (!has_radios) {
	// Nothing to report on wifi-less hardware.
	if (debug)
		print('{ "type": "OpenSoho", "radios": [] }\n');
	exit(0);
}

// dump-radios-caps.uc must never call exit() - it segfaults ucode when
// called from an include()'d file. It falls through to its end instead.
include('/usr/share/opensoho/dump-radios-caps.uc', { fs, wireless, debug, force });
