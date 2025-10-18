/*
	Based on the OpenWRT project LuCI - Lua Configuration Interface

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0
*/


'use strict';

var s = [0x0000, 0x0000, 0x0000, 0x0000];

function mul(a, b) {
	var r = [0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000];

	for (var j = 0; j < 4; j++) {
		var k = 0;
		for (var i = 0; i < 4; i++) {
			var t = a[i] * b[j] + r[i+j] + k;
			r[i+j] = t & 0xffff;
			k = t >>> 16;
		}
		r[j+4] = k;
	}

	r.length = 4;

	return r;
}

function add(a, n) {
	var r = [0x0000, 0x0000, 0x0000, 0x0000],
	    k = n;

	for (var i = 0; i < 4; i++) {
		var t = a[i] + k;
		r[i] = t & 0xffff;
		k = t >>> 16;
	}

	return r;
}

function shr(a, n) {
	var r = [a[0], a[1], a[2], a[3], 0x0000],
	    i = 4,
	    k = 0;

	for (; n > 16; n -= 16, i--)
		for (var j = 0; j < 4; j++)
			r[j] = r[j+1];

	for (; i > 0; i--) {
		var s = r[i-1];
		r[i-1] = (s >>> n) | k;
		k = ((s & ((1 << n) - 1)) << (16 - n));
	}

	r.length = 4;

	return r;
}

function s8(bytes, off) {
	var n = bytes[off];
	return (n > 0x7F) ? (n - 256) >>> 0 : n;
}

function u16(bytes, off) {
	return ((bytes[off + 1] << 8) + bytes[off]) >>> 0;
}

function sfh(s) {
	if (s === null || s.length === 0)
		return null;

	var bytes = [];

	for (var i = 0; i < s.length; i++) {
		var ch = s.charCodeAt(i);

		// Handle surrogate pairs
		if (ch >= 0xD800 && ch <= 0xDBFF && i + 1 < s.length) {
			const next = s.charCodeAt(i + 1);
			if (next >= 0xDC00 && next <= 0xDFFF) {
				ch = 0x10000 + ((ch - 0xD800) << 10) + (next - 0xDC00);
				i++;
			}
		}

		if (ch <= 0x7F)
			bytes.push(ch);
		else if (ch <= 0x7FF)
			bytes.push(((ch >>>  6) & 0x1F) | 0xC0,
			           ( ch         & 0x3F) | 0x80);
		else if (ch <= 0xFFFF)
			bytes.push(((ch >>> 12) & 0x0F) | 0xE0,
			           ((ch >>>  6) & 0x3F) | 0x80,
			           ( ch         & 0x3F) | 0x80);
		else if (ch <= 0x10FFFF)
			bytes.push(((ch >>> 18) & 0x07) | 0xF0,
			           ((ch >>> 12) & 0x3F) | 0x80,
			           ((ch >>   6) & 0x3F) | 0x80,
			           ( ch         & 0x3F) | 0x80);
	}

	if (!bytes.length)
		return null;

	var hash = (bytes.length >>> 0),
	    len = (bytes.length >>> 2),
	    off = 0, tmp;

	while (len--) {
		hash += u16(bytes, off);
		tmp   = ((u16(bytes, off + 2) << 11) ^ hash) >>> 0;
		hash  = ((hash << 16) ^ tmp) >>> 0;
		hash += hash >>> 11;
		off  += 4;
	}

	switch ((bytes.length & 3) >>> 0) {
	case 3:
		hash += u16(bytes, off);
		hash  = (hash ^ (hash << 16)) >>> 0;
		hash  = (hash ^ (s8(bytes, off + 2) << 18)) >>> 0;
		hash += hash >>> 11;
		break;

	case 2:
		hash += u16(bytes, off);
		hash  = (hash ^ (hash << 11)) >>> 0;
		hash += hash >>> 17;
		break;

	case 1:
		hash += s8(bytes, off);
		hash  = (hash ^ (hash << 10)) >>> 0;
		hash += hash >>> 1;
		break;
	}

	hash  = (hash ^ (hash << 3)) >>> 0;
	hash += hash >>> 5;
	hash  = (hash ^ (hash << 4)) >>> 0;
	hash += hash >>> 17;
	hash  = (hash ^ (hash << 25)) >>> 0;
	hash += hash >>> 6;

	return (0x100000000 + hash).toString(16).slice(1);
}

function toHex(n) {
  return n.toString(16).padStart(2, "0");
}

export default class PrngHelper{
	seed(n) {
		n = (n - 1)|0;
		s[0] = n & 0xffff;
		s[1] = n >>> 16;
		s[2] = 0;
		s[3] = 0;
	}

	int() {
		s = mul(s, [0x7f2d, 0x4c95, 0xf42d, 0x5851]);
		s = add(s, 1);

		var r = shr(s, 33);
		return (r[1] << 16) | r[0];
	}

	get() {
		var r = (this.int() % 0x7fffffff) / 0x7fffffff, l, u;

		switch (arguments.length) {
		case 0:
			return r;

		case 1:
			l = 1;
			u = arguments[0]|0;
			break;

		case 2:
			l = arguments[0]|0;
			u = arguments[1]|0;
			break;
		}

		return Math.floor(r * (u - l + 1)) + l;
	}

	derive_color(string) {
		this.seed(parseInt(sfh(string), 16));

		var r = this.get(128),
		    g = this.get(128),
		    min = 0,
		    max = 128;

		if ((r + g) < 128)
			min = 128 - r - g;
		else
			max = 255 - r - g;

		var b = min + Math.floor(this.get() * (max - min));

		//return '#%02x%02x%02x'.format(0xff - r, 0xff - g, 0xff - b);
		return `#${toHex(0xff - r)}${toHex(0xff - g)}${toHex(0xff - b)}`;
	}
};
