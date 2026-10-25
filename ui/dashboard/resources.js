import { loader, loading, openRecords } from "./charts.js";
import { healthyDeviceIds, selectAllLabel, selectDevices, setDevice } from "./selection.js";
import { byteStep, eventMarks, formatBytes, formatNumber, median, timeTicks } from "./timeseries.js";

// Remember the selected devices, columns and period across page loads.
const STORAGE_KEY = "dashboard_resource_history";

// Categorical palette validated on the light (#fff) and dark (#1c1c1c)
// surfaces. A device keeps its color slot while others are toggled, so at
// most one device per slot is shown.
const SERIES_COLORS = {
    light: ["#2a78d6", "#eb6834", "#1baf7a", "#eda100", "#e87ba4", "#008300", "#6250d6", "#e34948"],
    dark: ["#3987e5", "#d95926", "#199e70", "#c98500", "#d55181", "#008300", "#9085e9", "#e66767"],
};
const MAX_DEVICES = SERIES_COLORS.light.length;

// The color tells the devices apart, the dash pattern the columns.
const DASHES = [[], [6, 4], [2, 3], [10, 3, 2, 3], [14, 4]];

// Columns of one unit share a chart, so a chart never needs a second y axis.
const UNITS = [
    { key: "load", label: "Load", format: formatNumber },
    { key: "bytes", label: "Memory", format: formatBytes },
    { key: "count", label: "CPUs", format: formatNumber },
];

// The columns of /api/v1/device-resources (apiDeviceResources in opensoho.go).
const COLUMNS = [
    { key: "load1", label: "Load (1 min)", unit: "load" },
    { key: "load1_max", label: "Peak load (1 min)", unit: "load" },
    { key: "load5", label: "Load (5 min)", unit: "load" },
    { key: "load15", label: "Load (15 min)", unit: "load" },
    { key: "mem_used", label: "Memory used", unit: "bytes" },
    { key: "mem_available", label: "Memory available", unit: "bytes" },
    { key: "mem_free", label: "Memory free", unit: "bytes" },
    { key: "mem_cached", label: "Memory cached", unit: "bytes" },
    { key: "mem_buffered", label: "Memory buffered", unit: "bytes" },
    { key: "mem_total", label: "Memory total", unit: "bytes" },
    { key: "swap_used", label: "Swap used", unit: "bytes" },
    { key: "swap_free", label: "Swap free", unit: "bytes" },
    { key: "swap_total", label: "Swap total", unit: "bytes" },
    { key: "cpus", label: "CPUs", unit: "count" },
];

const RANGES = [
    { key: "1h", label: "Last hour", seconds: 3600 },
    { key: "6h", label: "Last 6 hours", seconds: 6 * 3600 },
    { key: "24h", label: "Last 24 hours", seconds: 24 * 3600 },
    { key: "7d", label: "Last 7 days", seconds: 7 * 24 * 3600 },
    { key: "30d", label: "Last 30 days", seconds: 30 * 24 * 3600 },
];

function loadPrefs() {
    try {
        const prefs = JSON.parse(window.localStorage?.getItem(STORAGE_KEY) || "null");
        if (prefs && Array.isArray(prefs.devices) && Array.isArray(prefs.columns)) {
            return { range: "24h", slots: {}, ...prefs };
        }
    } catch (_) {}
    return null;
}

function savePrefs(prefs) {
    try {
        window.localStorage?.setItem(STORAGE_KEY, JSON.stringify(prefs));
    } catch (_) {}
}

// Resolves a UI color variable (which may be a color-mix()) to a color the
// canvas understands.
function themeColor(name) {
    const probe = t.span({ style: `display:none; color: var(${name})` });
    document.body.appendChild(probe);
    const color = getComputedStyle(probe).color;
    probe.remove();
    return color;
}

function formatTime(ms, withDate) {
    return new Date(ms).toLocaleString(undefined, withDate
        ? { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" }
        : { hour: "2-digit", minute: "2-digit" });
}

function checkItem(label, checked, disabled, onchange) {
    const id = "resource_history_" + app.utils.randomString();

    return t.div(
        {
            className: "dropdown-item",
            onclick: (e) => {
                // workaround for clicking on the padded area
                e.target.querySelector("label")?.click();
            },
        },
        t.div(
            { className: "field" },
            t.input({
                type: "checkbox",
                className: "switch sm",
                id,
                checked,
                disabled,
                onchange: (e) => onchange(e.target.checked),
            }),
            t.label({ htmlFor: id }, label),
        ),
    );
}

// A toolbar button opening a list of check items.
function pickerMenu() {
    const text = t.span();
    const dropdown = t.div({ className: "dropdown sm nowrap gap-0 left resource-picker", popover: "auto" });

    const button = t.button(
        {
            type: "button",
            className: "btn sm secondary",
            popoverTargetElement: dropdown,
        },
        text,
        t.i({ className: "ri-arrow-down-s-line", ariaHidden: true }),
    );

    return {
        // the dropdown is not inside the button, where a click on an item
        // label would also toggle (close) the dropdown
        el: t.div({ className: "inline-flex" }, button, dropdown),
        update(label, items) {
            text.textContent = label;
            dropdown.replaceChildren(...items);
        },
    };
}

const EVENT_LABELS = { reboot: "rebooted" };

function eventLabel(mark) {
    return `${mark.device.name || mark.device.id} ${EVENT_LABELS[mark.type] || mark.type} at ${formatTime(mark.x, true)}`;
}

// Draws the events (see eventMarks) as dashed vertical lines with a small
// marker on top, and names them on hover.
const eventLinesPlugin = {
    id: "eventLines",
    afterDatasetsDraw(chart, _, options) {
        const { ctx, chartArea, scales } = chart;
        for (const mark of options.marks || []) {
            const x = scales.x.getPixelForValue(mark.x);
            if (x < chartArea.left || x > chartArea.right) continue;
            ctx.save();
            ctx.strokeStyle = mark.color;
            ctx.fillStyle = mark.color;
            ctx.lineWidth = 1.5;
            ctx.setLineDash([4, 3]);
            ctx.beginPath();
            ctx.moveTo(x, chartArea.top);
            ctx.lineTo(x, chartArea.bottom);
            ctx.stroke();
            ctx.setLineDash([]);
            ctx.beginPath();
            ctx.moveTo(x - 5, chartArea.top - 7);
            ctx.lineTo(x + 5, chartArea.top - 7);
            ctx.lineTo(x, chartArea.top);
            ctx.closePath();
            ctx.fill();
            ctx.restore();
        }
    },
    afterEvent(chart, args, options) {
        const event = args.event;
        if (event.type !== "mousemove" && event.type !== "mouseout") return;
        const near = event.type === "mousemove"
            ? (options.marks || []).filter((m) => Math.abs(chart.scales.x.getPixelForValue(m.x) - event.x) <= 5)
            : [];
        chart.canvas.title = near.map(eventLabel).join("\n");
    },
};

/**
 * Creates a line chart canvas for the columns of one unit.
 *
 * The chart itself is created when the canvas is mounted and destroyed when
 * it is unmounted.
 *
 * @param  {Object}  unit See UNITS.
 * @return {Element} The canvas, with an update({ datasets, marks, from, to }) method.
 */
function lineCanvas(unit) {
    let chart;
    let data = { datasets: [], marks: [], from: 0, to: 0 };

    const canvas = t.canvas({
        className: "resource-chart-canvas",
        role: "img",
        ariaLabel: `${unit.label} history chart, the values are listed in the table below`,
        onmount: (el) => {
            chart = new Chart(el, {
                type: "line",
                data: { labels: [], datasets: [] },
                plugins: [eventLinesPlugin],
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    animation: false,
                    parsing: false,
                    spanGaps: false,
                    interaction: { mode: "index", intersect: false },
                    // room for the event markers
                    layout: { padding: { top: 8 } },
                    elements: {
                        point: { radius: 0, hoverRadius: 4, hitRadius: 8 },
                        line: { borderWidth: 2, tension: 0 },
                    },
                    scales: {
                        x: {
                            type: "linear",
                            ticks: { maxRotation: 0 },
                            afterBuildTicks: (axis) => {
                                axis.ticks = timeTicks(data.from, data.to);
                            },
                        },
                        y: {
                            beginAtZero: true,
                            ticks: { maxTicksLimit: 6, callback: (v) => unit.format(v) },
                            afterDataLimits: (axis) => {
                                if (unit.key !== "bytes") return;
                                const step = byteStep(axis.max);
                                axis.max = Math.ceil(axis.max / step) * step || step;
                                axis.options.ticks.stepSize = step;
                            },
                            // the same width on every chart, so their time axes line up
                            afterFit: (axis) => {
                                axis.width = 64;
                            },
                        },
                    },
                    plugins: {
                        eventLines: { marks: [] },
                        legend: {
                            position: "bottom",
                            labels: { usePointStyle: true, pointStyle: "line", boxWidth: 28, padding: 16 },
                        },
                        tooltip: {
                            filter: (item) => item.parsed.y != null,
                            callbacks: {
                                title: (items) => items.length ? formatTime(items[0].parsed.x, true) : "",
                                label: (ctx) => ` ${ctx.dataset.label}: ${unit.format(ctx.parsed.y)}`,
                            },
                        },
                    },
                },
            });
            canvas.update(data);
        },
        onunmount: () => {
            chart?.destroy();
            chart = null;
        },
    });

    canvas.update = (next) => {
        data = next;
        if (!chart) return;

        const hint = themeColor("--surfaceTxtHintColor");
        const grid = themeColor("--surfaceAlt2Color");
        const withDate = data.to - data.from > 24 * 3600 * 1000;
        const { x, y } = chart.options.scales;

        x.min = data.from;
        x.max = data.to;
        x.ticks.color = hint;
        x.ticks.callback = (v) => formatTime(v, withDate);
        x.grid = { color: grid };
        y.ticks.color = hint;
        y.grid = { color: grid };
        chart.options.plugins.legend.labels.color = hint;

        chart.options.plugins.eventLines.marks = data.marks;
        chart.data.datasets = data.datasets;
        chart.update("none");
    };

    return canvas;
}

/**
 * Resource history card: the selected columns of the device_resources
 * collection over time, for the selected devices.
 *
 * @return {{el: Element, load: Function}}
 */
export function resourceHistoryChart() {
    const state = store({ isLoading: false });

    // undefined until the device list is known, then the defaults are picked
    let prefs = loadPrefs();
    let devices = [];
    let histories = {}; // device id -> points of the shown period
    let period = { from: 0, to: 0 };

    const devicePicker = pickerMenu();
    const columnPicker = pickerMenu();

    const rangeSelect = t.select(
        {
            onchange: (e) => {
                prefs.range = e.target.value;
                savePrefs(prefs);
                load();
            },
        },
        ...RANGES.map((r) => t.option({ value: r.key }, r.label)),
    );

    const charts = t.div({ className: "resource-charts" });
    const summary = t.div({ className: "net-scroll" });
    const chartsBlock = t.div(null, charts, summary);
    const content = t.div({ className: "resource-content" });
    let canvases = {};
    let shownUnits = "";

    let watcher;

    const el = t.div(
        {
            className: () => `resource-history ${state.isLoading ? "loading" : ""}`,
            onmount: () => {
                // the chart colors are set per color scheme
                watcher = watch(() => app.store.activeColorScheme, render);
            },
            onunmount: () => watcher?.unwatch(),
        },
        loader(state),
        t.div(
            { className: "net-toolbar" },
            devicePicker.el,
            columnPicker.el,
            t.label({ className: "inline-select" }, t.span(null, "Period"), rangeSelect),
        ),
        content,
    );

    function selectedDevices() {
        return devices.filter((d) => prefs.devices.includes(d.id));
    }

    function selectedColumns() {
        return COLUMNS.filter((c) => prefs.columns.includes(c.key));
    }

    function changeDevices(change) {
        change();
        savePrefs(prefs);
        renderPickers();
        load();
    }

    function selectAction(label, ids) {
        return t.button(
            {
                type: "button",
                className: "dropdown-item",
                disabled: !ids.length,
                onclick: () => changeDevices(() => selectDevices(prefs, ids, MAX_DEVICES)),
            },
            t.span({ className: "txt" }, selectAllLabel(label, ids.length, MAX_DEVICES)),
        );
    }

    function setColumn(key, on) {
        prefs.columns = prefs.columns.filter((c) => c !== key);
        if (on) {
            prefs.columns.push(key);
        }
        savePrefs(prefs);
        renderPickers();
        // every column of the selected devices is already loaded
        render();
    }

    function renderPickers() {
        const full = prefs.devices.length >= MAX_DEVICES;
        devicePicker.update(
            `Devices (${prefs.devices.length})`,
            devices.length
                ? [
                    selectAction("All devices", devices.map((d) => d.id)),
                    selectAction("All healthy devices", healthyDeviceIds(devices)),
                    t.hr(),
                    ...devices.map((d) => {
                        const on = prefs.devices.includes(d.id);
                        return checkItem(d.name || d.id, on, !on && full, (v) => {
                            changeDevices(() => setDevice(prefs, d.id, v, MAX_DEVICES));
                        });
                    }),
                ]
                : [t.div({ className: "dropdown-item" }, "No devices")],
        );
        columnPicker.update(
            `Metrics (${prefs.columns.length})`,
            COLUMNS.map((c) => checkItem(c.label, prefs.columns.includes(c.key), false, (v) => setColumn(c.key, v))),
        );
        rangeSelect.value = prefs.range;
    }

    function message(text) {
        content.replaceChildren(t.div({ className: "net-empty" }, text));
    }

    function deviceColor(device, scheme) {
        return SERIES_COLORS[scheme][prefs.slots[device.id] ?? 0];
    }

    // One series per device and column, on the union of all bucket times so
    // the tooltip lists all series at the hovered time. Missing buckets are
    // null, which breaks the line instead of drawing across the gap.
    function buildSeries(shownDevices, columns, scheme, bucket) {
        const times = new Set();
        for (const d of shownDevices) {
            for (const p of histories[d.id]?.points || []) {
                times.add(p.t);
            }
        }
        const sorted = [...times].sort((a, b) => a - b);
        const axis = [];
        sorted.forEach((t, i) => {
            if (i > 0 && t - sorted[i - 1] > 2 * bucket) {
                axis.push(sorted[i - 1] + bucket);
            }
            axis.push(t);
        });

        const series = [];
        for (const d of shownDevices) {
            const byTime = new Map((histories[d.id]?.points || []).map((p) => [p.t, p]));
            const color = deviceColor(d, scheme);
            columns.forEach((c) => {
                const values = (histories[d.id]?.points || []).map((p) => p[c.key]);
                series.push({
                    device: d,
                    column: c,
                    color,
                    latest: values.length ? values[values.length - 1] : null,
                    average: values.length ? values.reduce((a, b) => a + b, 0) / values.length : null,
                    median: median(values),
                    peak: values.length ? Math.max(...values) : null,
                    data: axis.map((t) => ({ x: t * 1000, y: byTime.get(t)?.[c.key] ?? null })),
                });
            });
        }
        return series;
    }

    function render() {
        if (!prefs) return;

        const shownDevices = selectedDevices();
        const columns = selectedColumns();
        if (!shownDevices.length) {
            return message("Select one or more devices.");
        }
        if (!columns.length) {
            return message("Select one or more metrics.");
        }
        if (!shownDevices.some((d) => histories[d.id]?.points?.length)) {
            return message("No samples in this period. Check that the history_retention_days setting is not 0.");
        }

        const units = UNITS.filter((u) => columns.some((c) => c.unit === u.key));
        const unitKeys = units.map((u) => u.key).join(",");
        if (unitKeys !== shownUnits) {
            shownUnits = unitKeys;
            canvases = {};
            charts.replaceChildren(...units.map((u) => {
                canvases[u.key] = lineCanvas(u);
                return t.div(
                    { className: "resource-chart" },
                    units.length > 1 ? t.div({ className: "band-label" }, u.label) : null,
                    t.div({ className: "resource-chart-wrap" }, canvases[u.key]),
                );
            }));
        }
        if (content.firstChild !== chartsBlock) {
            content.replaceChildren(chartsBlock);
        }

        const scheme = app.store.activeColorScheme === "dark" ? "dark" : "light";
        const bucket = Math.max(...shownDevices.map((d) => histories[d.id]?.bucket || 0), 30);
        const marks = eventMarks(shownDevices, histories, (d) => deviceColor(d, scheme));
        const rows = [];

        for (const u of units) {
            const unitColumns = columns.filter((c) => c.unit === u.key);
            const series = buildSeries(shownDevices, unitColumns, scheme, bucket);
            canvases[u.key].update({
                from: period.from * 1000,
                to: period.to * 1000,
                marks,
                datasets: series.map((s) => ({
                    label: `${s.device.name || s.device.id} · ${s.column.label}`,
                    data: s.data,
                    borderColor: s.color,
                    backgroundColor: s.color,
                    borderDash: DASHES[unitColumns.indexOf(s.column) % DASHES.length],
                })),
            });
            rows.push(...series.map((s) => ({ ...s, unit: u })));
        }

        summary.replaceChildren(
            t.table(
                { className: "net-table resource-summary" },
                t.thead(
                    null,
                    t.tr(
                        null,
                        t.th(null, "Device"),
                        t.th(null, "Metric"),
                        t.th({ className: "num" }, "Latest"),
                        t.th({ className: "num" }, "Average"),
                        t.th({ className: "num" }, "Median"),
                        t.th({ className: "num" }, "Peak"),
                    ),
                ),
                t.tbody(null, rows.map((r) => t.tr(
                    null,
                    t.td(
                        null,
                        t.span({ className: "legend-swatch", style: `background:${r.color}` }),
                        t.button(
                            {
                                type: "button",
                                className: "device-link",
                                title: "Show this device",
                                onclick: () => openRecords("devices", `id = "${r.device.id}"`),
                            },
                            r.device.name || r.device.id,
                        ),
                    ),
                    t.td(null, r.column.label),
                    t.td({ className: "num" }, r.unit.format(r.latest)),
                    t.td({ className: "num" }, r.unit.format(r.average)),
                    t.td({ className: "num" }, r.unit.format(r.median)),
                    t.td({ className: "num" }, r.unit.format(r.peak)),
                ))),
            ),
            // native replaceChildren() would render a null as text
            ...(marks.length
                ? [t.div(
                    { className: "resource-events" },
                    t.div({ className: "band-label" }, "Reboots"),
                    t.ul(null, marks.map((m) => t.li(
                        null,
                        t.span({ className: "legend-swatch", style: `background:${m.color}` }),
                        eventLabel(m),
                    ))),
                )]
                : []),
        );
    }

    async function load() {
        await loading(state, async () => {
            devices = await app.pb.collection("devices").getFullList({
                fields: "id,name,health_status",
                sort: "name",
                requestKey: "resource_history_devices",
            });

            if (!prefs) {
                prefs = { devices: [], columns: ["load1"], range: "24h", slots: {} };
                selectDevices(prefs, devices.map((d) => d.id), MAX_DEVICES);
                savePrefs(prefs);
            }
            // forget devices that were deleted
            const known = new Set(devices.map((d) => d.id));
            prefs.devices = prefs.devices.filter((id) => known.has(id));
            if (!RANGES.some((r) => r.key === prefs.range)) {
                prefs.range = "24h";
            }
            renderPickers();

            const range = RANGES.find((r) => r.key === prefs.range);
            const to = Math.floor(Date.now() / 1000);
            const from = to - range.seconds;
            const results = await Promise.all(prefs.devices.map((id) => {
                return app.pb.send(`/api/v1/device-resources/${encodeURIComponent(id)}`, {
                    method: "GET",
                    query: { from, to },
                    requestKey: `resource_history_${id}`,
                });
            }));

            histories = {};
            prefs.devices.forEach((id, i) => {
                histories[id] = results[i];
            });
            period = { from, to };
        });

        // after loading, so that newly shown elements don't mount mid-load
        render();
    }

    return { el, load };
}
