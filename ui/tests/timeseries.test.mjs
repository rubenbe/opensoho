// Unit tests of ui/dashboard/timeseries.js, run with: node --test ui/tests/
//
// The time zone is switched through process.env.TZ, which node applies
// immediately, so each test sets the zone it needs.

import assert from "node:assert/strict";
import { afterEach, describe, test } from "node:test";

import { byteStep, formatBytes, formatNumber, median, timeTicks } from "../dashboard/timeseries.js";

const originalTZ = process.env.TZ;
afterEach(() => {
    if (originalTZ === undefined) {
        delete process.env.TZ;
    } else {
        process.env.TZ = originalTZ;
    }
});

const MiB = 1024 ** 2;
const GiB = 1024 ** 3;

// "YYYY-MM-DD HH:MM" in the current local time zone
function local(ms) {
    const d = new Date(ms);
    const pad = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function ticks(tz, from, to) {
    process.env.TZ = tz;
    return timeTicks(Date.parse(from), Date.parse(to)).map((t) => t.value);
}

describe("timeTicks", () => {
    test("every 3 hours across the switch to winter time (Brussels)", () => {
        const values = ticks("Europe/Brussels", "2026-10-24T09:20:00Z", "2026-10-25T09:20:00Z");
        assert.deepEqual(values.map(local), [
            "2026-10-24 12:00", "2026-10-24 15:00", "2026-10-24 18:00", "2026-10-24 21:00",
            "2026-10-25 00:00", "2026-10-25 03:00", "2026-10-25 06:00", "2026-10-25 09:00",
        ]);
        // the night of the switch has an extra hour
        assert.equal(values[5] - values[4], 4 * 3600 * 1000);
    });

    test("every 3 hours across the switch to summer time (Brussels)", () => {
        const values = ticks("Europe/Brussels", "2026-03-28T09:20:00Z", "2026-03-29T09:20:00Z");
        assert.deepEqual(values.map(local), [
            "2026-03-28 12:00", "2026-03-28 15:00", "2026-03-28 18:00", "2026-03-28 21:00",
            "2026-03-29 00:00", "2026-03-29 03:00", "2026-03-29 06:00", "2026-03-29 09:00",
        ]);
        // the night of the switch misses an hour
        assert.equal(values[5] - values[4], 2 * 3600 * 1000);
    });

    test("every hour through the repeated hour (Brussels)", () => {
        const values = ticks("Europe/Brussels", "2026-10-24T22:10:00Z", "2026-10-25T04:10:00Z");
        // 02:00 happens twice that night, it is only labeled once
        assert.deepEqual(values.map(local), [
            "2026-10-25 01:00", "2026-10-25 02:00", "2026-10-25 03:00", "2026-10-25 04:00", "2026-10-25 05:00",
        ]);
        assert.deepEqual(values.map((v) => new Date(v).toISOString()), [
            "2026-10-24T23:00:00.000Z", "2026-10-25T00:00:00.000Z", "2026-10-25T02:00:00.000Z",
            "2026-10-25T03:00:00.000Z", "2026-10-25T04:00:00.000Z",
        ]);
    });

    test("daily at local midnight across the switch to winter time (Brussels)", () => {
        const values = ticks("Europe/Brussels", "2026-10-20T10:00:00Z", "2026-10-27T10:00:00Z");
        assert.deepEqual(values.map(local), [
            "2026-10-21 00:00", "2026-10-22 00:00", "2026-10-23 00:00", "2026-10-24 00:00",
            "2026-10-25 00:00", "2026-10-26 00:00", "2026-10-27 00:00",
        ]);
    });

    test("every 3 hours across the switch to winter time (New York)", () => {
        const values = ticks("America/New_York", "2026-10-31T16:20:00Z", "2026-11-01T16:20:00Z");
        assert.deepEqual(values.map(local), [
            "2026-10-31 15:00", "2026-10-31 18:00", "2026-10-31 21:00", "2026-11-01 00:00",
            "2026-11-01 03:00", "2026-11-01 06:00", "2026-11-01 09:00",
        ]);
        assert.equal(values[4] - values[3], 4 * 3600 * 1000);
    });

    test("every 15 minutes with a +05:45 offset (Kathmandu)", () => {
        const values = ticks("Asia/Kathmandu", "2026-10-25T10:07:00Z", "2026-10-25T11:07:00Z");
        assert.deepEqual(values.map(local), [
            "2026-10-25 16:00", "2026-10-25 16:15", "2026-10-25 16:30", "2026-10-25 16:45",
        ]);
    });

    test("every hour with a +05:45 offset (Kathmandu)", () => {
        const values = ticks("Asia/Kathmandu", "2026-10-25T10:07:00Z", "2026-10-25T16:07:00Z");
        assert.deepEqual(values.map(local), [
            "2026-10-25 16:00", "2026-10-25 17:00", "2026-10-25 18:00",
            "2026-10-25 19:00", "2026-10-25 20:00", "2026-10-25 21:00",
        ]);
    });

    test("weekly at midnight over 30 days (UTC)", () => {
        const values = ticks("UTC", "2026-09-25T10:07:00Z", "2026-10-25T10:07:00Z");
        assert.deepEqual(values.map(local), [
            "2026-09-26 00:00", "2026-10-03 00:00", "2026-10-10 00:00", "2026-10-17 00:00", "2026-10-24 00:00",
        ]);
    });

    // the periods the dashboard offers, ending right after each switch
    const periods = [3600, 6 * 3600, 24 * 3600, 7 * 24 * 3600, 30 * 24 * 3600];
    const ends = ["2026-10-25T09:20:00Z", "2026-03-29T09:20:00Z", "2026-11-01T16:20:00Z", "2026-07-01T12:34:56Z"];

    for (const tz of ["UTC", "Europe/Brussels", "America/New_York", "Asia/Kathmandu", "Australia/Lord_Howe"]) {
        test(`round, ordered ticks inside the period (${tz})`, () => {
            for (const end of ends) {
                for (const seconds of periods) {
                    const to = Date.parse(end);
                    const from = to - seconds * 1000;
                    process.env.TZ = tz;
                    const values = timeTicks(from, to).map((t) => t.value);
                    const label = `${tz} ${seconds}s up to ${end}`;

                    assert.ok(values.length >= 3 && values.length <= 8, `${label}: ${values.length} ticks`);
                    values.forEach((v, i) => {
                        assert.ok(v >= from && v <= to, `${label}: ${local(v)} outside the period`);
                        if (i > 0) {
                            assert.ok(v > values[i - 1], `${label}: ${local(v)} not after ${local(values[i - 1])}`);
                        }
                        const d = new Date(v);
                        if (seconds >= 6 * 3600) {
                            assert.equal(d.getMinutes(), 0, `${label}: ${local(v)} not on the hour`);
                        }
                        if (seconds >= 7 * 24 * 3600) {
                            assert.equal(d.getHours(), 0, `${label}: ${local(v)} not at midnight`);
                        }
                    });
                }
            }
        });
    }
});

describe("median", () => {
    test("no values", () => assert.equal(median([]), null));
    test("odd count", () => assert.equal(median([5, 1, 3]), 3));
    test("even count", () => assert.equal(median([4, 1, 3, 2]), 2.5));
    test("ignores a spike", () => assert.equal(median([0.1, 0.1, 0.1, 5]), 0.1));
    test("doesn't reorder the input", () => {
        const values = [3, 1, 2];
        median(values);
        assert.deepEqual(values, [3, 1, 2]);
    });
});

describe("byteStep", () => {
    test("small values", () => {
        assert.equal(byteStep(0), 1);
        assert.equal(byteStep(5), 1);
        assert.equal(byteStep(6), 2);
    });
    test("binary steps of at most 5 ticks", () => {
        assert.equal(byteStep(300 * MiB), 64 * MiB);
        assert.equal(byteStep(381 * MiB), 128 * MiB);
        assert.equal(byteStep(GiB), 256 * MiB);
        assert.equal(byteStep(4 * GiB), GiB);
    });
});

describe("formatBytes", () => {
    test("missing value", () => assert.equal(formatBytes(null), "—"));
    test("bytes", () => {
        assert.equal(formatBytes(0), "0 B");
        assert.equal(formatBytes(1023), "1023 B");
    });
    test("binary units", () => {
        assert.equal(formatBytes(1024), "1.0 KiB");
        assert.equal(formatBytes(1536), "1.5 KiB");
        assert.equal(formatBytes(64 * MiB), "64.0 MiB");
        assert.equal(formatBytes(100 * MiB), "100 MiB");
        assert.equal(formatBytes(2 * GiB), "2.0 GiB");
    });
});

describe("formatNumber", () => {
    test("missing value", () => assert.equal(formatNumber(null), "—"));
    test("two decimals at most", () => {
        assert.equal(formatNumber(0.333333), "0.33");
        assert.equal(formatNumber(2), "2");
        assert.equal(formatNumber(0.8), "0.8");
    });
});
