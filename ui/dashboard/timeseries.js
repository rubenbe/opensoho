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

// "5" -> "5 GHz"; an unknown band is empty.
export function bandLabel(band) {
    return band ? `${band} GHz` : "";
}

// Where a client was last: "Living Room AP · 5 GHz", or whichever is known.
export function clientLocation(client) {
    return [client.device_name, bandLabel(client.band)].filter(Boolean).join(" · ");
}

// A signal strength (RSSI), rounded to whole dBm.
export function formatDbm(v) {
    return v == null ? "—" : `${Math.round(v)} dBm`;
}

/**
 * Groups the summary rows per item, in the order of the items, and the rows
 * of an item in the order of the metrics.
 *
 * @param  {Array} rows    [{ item, metric, ... }]
 * @param  {Array} items   [{ id }] in display order.
 * @param  {Array} metrics [{ key }] in display order.
 * @return {Array} [{ item, rows }] without the items that have no rows.
 */
export function groupRows(rows, items, metrics) {
    const metricIndex = (row) => metrics.findIndex((m) => m.key === row.metric.key);
    return items
        .map((item) => ({
            item,
            rows: rows
                .filter((r) => r.item.id === item.id)
                .sort((a, b) => metricIndex(a) - metricIndex(b)),
        }))
        .filter((group) => group.rows.length);
}

// Identifies an event mark (see eventMarks).
export function markKey(mark) {
    return `${mark.entity.id}|${mark.type}|${mark.x}`;
}

/**
 * The marks on the line nearest to a pointer or tap position, if that line is
 * within tolerance pixels. Lines drawn at the same position (e.g. the reboot
 * of an AP several clients used) are found together.
 *
 * @param  {Array}    marks     See eventMarks.
 * @param  {Function} pixelOf   Mark -> its x position in pixels.
 * @param  {number}   x         The pointer position in pixels.
 * @param  {number}   tolerance Pixels.
 * @return {Array}    The marks, or [].
 */
export function marksNear(marks, pixelOf, x, tolerance) {
    let nearest = null;
    for (const mark of marks) {
        const distance = Math.abs(pixelOf(mark) - x);
        if (distance <= tolerance && (nearest === null || distance < Math.abs(nearest - x))) {
            nearest = pixelOf(mark);
        }
    }
    return nearest === null ? [] : marks.filter((m) => Math.abs(pixelOf(m) - nearest) < 1);
}

// The newest n marks (see eventMarks), newest first.
export function latestEvents(marks, n) {
    return [...marks].sort((a, b) => b.x - a.x).slice(0, n);
}

/**
 * The events next to the selected ones in time: those at the nearest time
 * after (direction 1) or before (direction -1) them. Events at the same time
 * (e.g. the reboot of an AP several clients used) come together, as they are
 * drawn as one line.
 *
 * @param  {Array}  marks     See eventMarks.
 * @param  {Array}  selected  The selected marks.
 * @param  {number} direction 1 for the next events, -1 for the previous ones.
 * @return {Array}  The marks, or [] when there are none.
 */
export function adjacentEvents(marks, selected, direction) {
    if (!selected.length) {
        return [];
    }
    const from = direction > 0 ? Math.max(...selected.map((m) => m.x)) : Math.min(...selected.map((m) => m.x));
    let nearest = null;
    for (const mark of marks) {
        if ((mark.x - from) * direction > 0 && (nearest === null || Math.abs(mark.x - from) < Math.abs(nearest - from))) {
            nearest = mark.x;
        }
    }
    return nearest === null ? [] : marks.filter((m) => m.x === nearest);
}
