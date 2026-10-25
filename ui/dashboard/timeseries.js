// Pure helpers of the resource history chart, without DOM or Chart.js, so
// they can be unit tested with node (see ui/tests).

export function formatNumber(v) {
    return v == null ? "—" : String(Math.round(v * 100) / 100);
}

export function formatBytes(v) {
    if (v == null) return "—";
    const units = ["B", "KiB", "MiB", "GiB", "TiB"];
    let i = 0;
    while (Math.abs(v) >= 1024 && i < units.length - 1) {
        v /= 1024;
        i++;
    }
    return `${v.toFixed(i === 0 || v >= 100 ? 0 : 1)} ${units[i]}`;
}

// A round binary step (1, 2, 4, ... 512 KiB/MiB/GiB) giving at most 5 ticks.
export function byteStep(max) {
    for (const base of [1, 1024, 1024 ** 2, 1024 ** 3, 1024 ** 4]) {
        for (const m of [1, 2, 4, 8, 16, 32, 64, 128, 256, 512]) {
            if (max / (base * m) <= 5) return base * m;
        }
    }
    return 1024 ** 5;
}

export function median(values) {
    if (!values.length) return null;
    const sorted = [...values].sort((a, b) => a - b);
    const mid = Math.floor(sorted.length / 2);
    return sorted.length % 2 ? sorted[mid] : (sorted[mid - 1] + sorted[mid]) / 2;
}

const HOUR = 3600 * 1000;
const TICK_STEPS = [5, 15, 30, 60, 120, 180, 360, 720, 1440, 2880, 10080].map((m) => m * 60 * 1000);

// Time axis ticks on round local times (e.g. every 3 hours at 00:00, 03:00, ...).
export function timeTicks(from, to, maxTicks = 8) {
    const step = TICK_STEPS.find((s) => (to - from) / s <= maxTicks) || TICK_STEPS[TICK_STEPS.length - 1];
    const ticks = [];

    if (step < HOUR) {
        // align to the local time, not UTC; a DST switch moves it by whole
        // hours, which keeps sub-hour steps round
        const offset = new Date(from).getTimezoneOffset() * 60 * 1000;
        for (let v = Math.ceil((from - offset) / step) * step + offset; v <= to; v += step) {
            ticks.push({ value: v });
        }
        return ticks;
    }

    // walk the wall clock, so the ticks stay on round hours across a DST switch
    const hours = step / HOUR;
    const advance = (d, n) => hours >= 24 ? d.setDate(d.getDate() + n / 24) : d.setHours(d.getHours() + n);
    const d = new Date(from);
    d.setMinutes(0, 0, 0);
    if (hours >= 24) {
        d.setHours(0);
    }
    while (d.getTime() < from || (hours < 24 && d.getHours() % hours !== 0)) {
        advance(d, hours >= 24 ? 24 : 1);
    }
    while (d.getTime() <= to) {
        ticks.push({ value: d.getTime() });
        const previous = d.getTime();
        advance(d, hours);
        if (d.getTime() <= previous) {
            d.setTime(previous + step);
        }
    }
    return ticks;
}

/**
 * The events (reboots, roams, ...) of the shown items, as marks for the charts.
 *
 * @param  {Array}    items     [{ id, ... }] in display order.
 * @param  {Object}   histories Item id -> API response with events [{ type, t, ... }].
 * @param  {Function} colorOf   Item -> color.
 * @return {Array}    [{ x, color, entity, type, event }] sorted by time; x in milliseconds.
 */
export function eventMarks(items, histories, colorOf) {
    const marks = [];
    for (const entity of items) {
        for (const event of histories[entity.id]?.events || []) {
            marks.push({ x: event.t * 1000, color: colorOf(entity), entity, type: event.type, event });
        }
    }
    return marks.sort((a, b) => a.x - b.x);
}

// Formats bytes per second as bits per second (network rates are in bits).
export function formatBitRate(bytesPerSecond) {
    if (bytesPerSecond == null) return "—";
    const units = ["bit/s", "kbit/s", "Mbit/s", "Gbit/s"];
    let v = bytesPerSecond * 8;
    let i = 0;
    while (Math.abs(v) >= 1000 && i < units.length - 1) {
        v /= 1000;
        i++;
    }
    return `${v.toFixed(i === 0 || v >= 100 ? 0 : 1)} ${units[i]}`;
}

// A round step (1, 2 or 5 times a power of 10 bit/s) giving at most 5 ticks
// up to max, in bytes per second like the values.
export function bitRateStep(max) {
    const bits = max * 8;
    for (let power = 1; ; power *= 10) {
        for (const m of [1, 2, 5]) {
            if (bits / (m * power) <= 5) return (m * power) / 8;
        }
    }
}
