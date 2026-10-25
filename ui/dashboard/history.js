import { loader, loading } from "./charts.js";
import { selectAllLabel, selectItems, setItem } from "./selection.js";
import { eventMarks, median, timeTicks } from "./timeseries.js";

// A dashboard card charting the history of some items (devices, clients)
// over time: one line per selected item and metric, one chart per unit.
// resources.js and clients.js configure it.

// Categorical palette validated on the light (#fff) and dark (#1c1c1c)
// surfaces. An item keeps its color slot while others are toggled, so at most
// one item per slot is shown.
const SERIES_COLORS = {
    light: ["#2a78d6", "#eb6834", "#1baf7a", "#eda100", "#e87ba4", "#008300", "#6250d6", "#e34948"],
    dark: ["#3987e5", "#d95926", "#199e70", "#c98500", "#d55181", "#008300", "#9085e9", "#e66767"],
};
export const MAX_ITEMS = SERIES_COLORS.light.length;

// The color tells the items apart, the dash pattern the metrics.
const DASHES = [[], [6, 4], [2, 3], [10, 3, 2, 3], [14, 4]];

const RANGES = [
    { key: "1h", label: "Last hour", seconds: 3600 },
    { key: "6h", label: "Last 6 hours", seconds: 6 * 3600 },
    { key: "24h", label: "Last 24 hours", seconds: 24 * 3600 },
    { key: "7d", label: "Last 7 days", seconds: 7 * 24 * 3600 },
    { key: "30d", label: "Last 30 days", seconds: 30 * 24 * 3600 },
];

function loadPrefs(storageKey) {
    try {
        const prefs = JSON.parse(window.localStorage?.getItem(storageKey) || "null");
        if (prefs && !prefs.selected && Array.isArray(prefs.devices)) {
            // saved before clients had a history card
            prefs.selected = prefs.devices;
            delete prefs.devices;
        }
        if (prefs && Array.isArray(prefs.selected) && Array.isArray(prefs.columns)) {
            return { range: "24h", slots: {}, ...prefs };
        }
    } catch (_) {}
    return null;
}

function savePrefs(storageKey, prefs) {
    try {
        window.localStorage?.setItem(storageKey, JSON.stringify(prefs));
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

export function formatTime(ms, withDate) {
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

// Draws the events (see eventMarks) as dashed vertical lines with a small
// marker on top, and names them (options.label) on hover.
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
        chart.canvas.title = near.map((m) => options.label(m)).join("\n");
    },
};

/**
 * Creates a line chart canvas for the metrics of one unit.
 *
 * The chart itself is created when the canvas is mounted and destroyed when
 * it is unmounted.
 *
 * @param  {Object}   unit       { key, label, format(value), step?(max) }
 * @param  {Function} eventLabel Mark -> text shown on hover.
 * @return {Element}  The canvas, with an update({ datasets, marks, from, to }) method.
 */
function lineCanvas(unit, eventLabel) {
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
                                if (!unit.step) return;
                                const step = unit.step(axis.max);
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
                        eventLines: { marks: [], label: eventLabel },
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
 * @param  {Object}   config
 * @param  {string}   config.storageKey     localStorage key of the view preferences.
 * @param  {string}   config.requestKey     Prefix of the request keys.
 * @param  {Object}   config.names          { item: "Device", items: "Devices", none: "No devices" }
 * @param  {Array}    config.units          [{ key, label, format(value), step?(max) }]
 * @param  {Array}    config.metrics        [{ key, label, unit }], the keys of the history points.
 * @param  {Array}    config.defaultMetrics Metric keys shown at first.
 * @param  {Function} config.loadItems      async ({ from, to }) -> [{ id, ... }] in display order.
 * @param  {Function} config.itemName       Item -> name.
 * @param  {Function} [config.itemDetail]   Item -> short text shown after its name in the table.
 * @param  {Function} config.defaultItems   Items -> ids shown at first.
 * @param  {Function} config.selectActions  Items -> [{ label, ids }], "select all ..." shortcuts.
 * @param  {Function} config.loadHistory    async (id, { from, to }) -> { bucket, points: [{ t, ... }], events: [{ type, t }] }
 * @param  {Function} config.openItem       Item -> void, opens its record.
 * @param  {string}   config.openTitle      Hover text of the item links.
 * @param  {Function} config.eventLabel     (mark, itemName) -> text, see eventMarks.
 * @param  {string}   config.eventsTitle    Heading of the event list.
 * @param  {string}   config.emptyMessage   Shown when there are no samples.
 * @return {{el: Element, load: Function}}
 */
export function historyChart(config) {
    const state = store({ isLoading: false });

    // null until the items are known, then the defaults are picked
    let prefs = loadPrefs(config.storageKey);
    let items = [];
    let histories = {}; // item id -> history of the shown period
    let period = { from: 0, to: 0 };

    const itemPicker = pickerMenu();
    const metricPicker = pickerMenu();

    const rangeSelect = t.select(
        {
            onchange: (e) => {
                prefs.range = e.target.value;
                savePrefs(config.storageKey, prefs);
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
            itemPicker.el,
            metricPicker.el,
            t.label({ className: "inline-select" }, t.span(null, "Period"), rangeSelect),
        ),
        content,
    );

    const nameOf = (item) => config.itemName(item);

    const eventLabel = (mark) => config.eventLabel(mark, nameOf(mark.entity));

    function selectedItems() {
        return items.filter((i) => prefs.selected.includes(i.id));
    }

    function selectedMetrics() {
        return config.metrics.filter((m) => prefs.columns.includes(m.key));
    }

    function changeItems(change) {
        change();
        savePrefs(config.storageKey, prefs);
        renderPickers();
        load();
    }

    function selectAction(label, ids) {
        return t.button(
            {
                type: "button",
                className: "dropdown-item",
                disabled: !ids.length,
                onclick: () => changeItems(() => selectItems(prefs, ids, MAX_ITEMS)),
            },
            t.span({ className: "txt" }, selectAllLabel(label, ids.length, MAX_ITEMS)),
        );
    }

    function setMetric(key, on) {
        prefs.columns = prefs.columns.filter((c) => c !== key);
        if (on) {
            prefs.columns.push(key);
        }
        savePrefs(config.storageKey, prefs);
        renderPickers();
        // every metric of the selected items is already loaded
        render();
    }

    function renderPickers() {
        const full = prefs.selected.length >= MAX_ITEMS;
        const actions = config.selectActions(items);
        itemPicker.update(
            `${config.names.items} (${prefs.selected.length})`,
            items.length
                ? [
                    ...actions.map((a) => selectAction(a.label, a.ids)),
                    ...(actions.length ? [t.hr()] : []),
                    ...items.map((i) => {
                        const on = prefs.selected.includes(i.id);
                        return checkItem(nameOf(i), on, !on && full, (v) => {
                            changeItems(() => setItem(prefs, i.id, v, MAX_ITEMS));
                        });
                    }),
                ]
                : [t.div({ className: "dropdown-item" }, config.names.none)],
        );
        metricPicker.update(
            `Metrics (${prefs.columns.length})`,
            config.metrics.map((m) => checkItem(m.label, prefs.columns.includes(m.key), false, (v) => setMetric(m.key, v))),
        );
        rangeSelect.value = prefs.range;
    }

    function message(text) {
        content.replaceChildren(t.div({ className: "net-empty" }, text));
    }

    function itemColor(item, scheme) {
        return SERIES_COLORS[scheme][prefs.slots[item.id] ?? 0];
    }

    // One series per item and metric, on the union of all bucket times so
    // the tooltip lists all series at the hovered time. Missing buckets are
    // null, which breaks the line instead of drawing across the gap.
    function buildSeries(shownItems, metrics, scheme, bucket) {
        const times = new Set();
        for (const i of shownItems) {
            for (const p of histories[i.id]?.points || []) {
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
        for (const item of shownItems) {
            const points = histories[item.id]?.points || [];
            const byTime = new Map(points.map((p) => [p.t, p]));
            const color = itemColor(item, scheme);
            metrics.forEach((m) => {
                const values = points.map((p) => p[m.key]).filter((v) => v != null);
                series.push({
                    item,
                    metric: m,
                    color,
                    latest: values.length ? values[values.length - 1] : null,
                    average: values.length ? values.reduce((a, b) => a + b, 0) / values.length : null,
                    median: median(values),
                    peak: values.length ? Math.max(...values) : null,
                    data: axis.map((t) => ({ x: t * 1000, y: byTime.get(t)?.[m.key] ?? null })),
                });
            });
        }
        return series;
    }

    function render() {
        if (!prefs) return;

        const shownItems = selectedItems();
        const metrics = selectedMetrics();
        if (!shownItems.length) {
            return message(`Select one or more ${config.names.items.toLowerCase()}.`);
        }
        if (!metrics.length) {
            return message("Select one or more metrics.");
        }
        if (!shownItems.some((i) => histories[i.id]?.points?.length)) {
            return message(config.emptyMessage);
        }

        const units = config.units.filter((u) => metrics.some((m) => m.unit === u.key));
        const unitKeys = units.map((u) => u.key).join(",");
        if (unitKeys !== shownUnits) {
            shownUnits = unitKeys;
            canvases = {};
            charts.replaceChildren(...units.map((u) => {
                canvases[u.key] = lineCanvas(u, eventLabel);
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
        const bucket = Math.max(...shownItems.map((i) => histories[i.id]?.bucket || 0), 30);
        const marks = eventMarks(shownItems, histories, (i) => itemColor(i, scheme));
        const rows = [];

        for (const u of units) {
            const unitMetrics = metrics.filter((m) => m.unit === u.key);
            const series = buildSeries(shownItems, unitMetrics, scheme, bucket);
            canvases[u.key].update({
                from: period.from * 1000,
                to: period.to * 1000,
                marks,
                datasets: series.map((s) => ({
                    label: `${nameOf(s.item)} · ${s.metric.label}`,
                    data: s.data,
                    borderColor: s.color,
                    backgroundColor: s.color,
                    borderDash: DASHES[unitMetrics.indexOf(s.metric) % DASHES.length],
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
                        t.th(null, config.names.item),
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
                                title: config.openTitle,
                                onclick: () => config.openItem(r.item),
                            },
                            nameOf(r.item),
                        ),
                        config.itemDetail?.(r.item)
                            ? t.span({ className: "resource-item-detail" }, ` · ${config.itemDetail(r.item)}`)
                            : null,
                    ),
                    t.td(null, r.metric.label),
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
                    t.div({ className: "band-label" }, config.eventsTitle),
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
            const range = RANGES.find((r) => r.key === prefs?.range) || RANGES.find((r) => r.key === "24h");
            const to = Math.floor(Date.now() / 1000);
            const from = to - range.seconds;

            items = await config.loadItems({ from, to });

            if (!prefs) {
                prefs = { selected: [], columns: [...config.defaultMetrics], range: range.key, slots: {} };
                selectItems(prefs, config.defaultItems(items), MAX_ITEMS);
                savePrefs(config.storageKey, prefs);
            }
            // forget items that were deleted
            const known = new Set(items.map((i) => i.id));
            prefs.selected = prefs.selected.filter((id) => known.has(id));
            prefs.range = range.key;
            renderPickers();

            const results = await Promise.all(prefs.selected.map((id) => config.loadHistory(id, { from, to })));

            histories = {};
            prefs.selected.forEach((id, i) => {
                histories[id] = results[i];
            });
            period = { from, to };
        });

        // after loading, so that newly shown elements don't mount mid-load
        render();
    }

    return { el, load };
}
