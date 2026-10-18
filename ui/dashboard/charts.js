// Chart.js based building blocks shared by the dashboard cards.

export const PALETTE = [
    "#4e79a7", "#f28e2b", "#e15759", "#76b7b2", "#59a14f",
    "#edc948", "#b07aa1", "#ff9da7", "#9c755f", "#bab0ac",
];

let chartJsPromise;

/**
 * Loads the vendored Chart.js bundle (once).
 *
 * @return {Promise}
 */
export function loadChartJs() {
    if (!chartJsPromise) {
        chartJsPromise = new Promise((resolve, reject) => {
            document.head.appendChild(t.script({
                src: app.opensoho.baseURL + "libs/chart.umd.min.js",
                onload: resolve,
                onerror: reject,
            }));
        });
    }

    return chartJsPromise;
}

/**
 * Navigates to the records list of a collection, filtered.
 *
 * @param {string} collection
 * @param {string} filter
 */
export function openRecords(collection, filter) {
    // the collections page resets the filter when the active collection
    // changes, which includes switching from the one that was last active
    app.store.activeCollection = collection;
    window.location.hash = `#/collections?collection=${collection}&filter=${encodeURIComponent(filter)}`;
}

/**
 * Calls loadFunc, reporting API errors (except aborted requests).
 *
 * @param {Object}   state    Store with an isLoading field.
 * @param {Function} loadFunc
 */
export async function loading(state, loadFunc) {
    state.isLoading = true;
    try {
        await loadFunc();
    } catch (err) {
        if (!err?.isAbort) {
            app.checkApiError(err);
        }
    } finally {
        state.isLoading = false;
    }
}

export function loader(state) {
    return t.span({ hidden: () => !state.isLoading, className: "loader chart-loader" });
}

/**
 * Creates a pie chart canvas.
 *
 * The chart itself is created when the canvas is mounted and destroyed when
 * it is unmounted.
 *
 * @param  {Object}   options
 * @param  {boolean}  [options.legend]  Show the Chart.js legend below the pie.
 * @param  {Function} [options.onclick] Called with the clicked slice.
 * @return {Element}  The canvas, with an update(slices) method.
 *                    Slices: [{ label, value, color, ... }]
 */
export function pieCanvas({ legend = true, onclick } = {}) {
    let chart;
    let slices = [];

    const canvas = t.canvas({
        className: "chart-canvas",
        onmount: (el) => {
            chart = new Chart(el, {
                type: "pie",
                data: {
                    labels: [],
                    datasets: [{ data: [], backgroundColor: [], borderWidth: 0 }],
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    onClick: (_, elements) => {
                        if (!elements.length) return;
                        const slice = slices[elements[0].index];
                        if (slice) {
                            onclick?.(slice);
                        }
                    },
                    plugins: {
                        legend: legend
                            ? {
                                position: "bottom",
                                labels: { color: "#617079", boxWidth: 12, padding: 16 },
                            }
                            : { display: false },
                        tooltip: {
                            callbacks: {
                                label: (ctx) => ` ${ctx.label}: ${ctx.parsed}`,
                            },
                        },
                    },
                },
            });
            canvas.update(slices);
        },
        onunmount: () => {
            chart?.destroy();
            chart = null;
        },
    });

    canvas.update = (next) => {
        slices = next;
        if (!chart) return;
        chart.data.labels = slices.map((s) => s.label);
        chart.data.datasets[0].data = slices.map((s) => s.value);
        chart.data.datasets[0].backgroundColor = slices.map((s) => s.color);
        chart.update();
    };

    return canvas;
}

/**
 * A single pie chart with a loader, at the standard card chart height.
 *
 * @param  {Object}   state   Store with an isLoading field.
 * @param  {Element}  canvas  See pieCanvas().
 * @return {Element}
 */
export function pieChart(state, canvas) {
    return t.div(
        { className: () => `chart-wrapper ${state.isLoading ? "loading" : ""}` },
        loader(state),
        canvas,
    );
}

/**
 * One small pie per frequency band, side by side, with a HTML legend so the
 * pies always have the same size and line up.
 *
 * @param  {Object}   state      Store with an isLoading field.
 * @param  {Function} onselect   Called with (band, slice).
 * @return {Element}  The element, with an update(series) method.
 *                    Series: [{ key, label, slices: [{ label, value, color, ... }] }]
 */
export function bandPieCharts(state, onselect) {
    const columns = t.div({ className: "band-charts-columns" });
    let canvases = {};

    const el = t.div(
        { className: () => `band-charts ${state.isLoading ? "loading" : ""}` },
        loader(state),
        columns,
    );

    el.update = (series) => {
        const keys = series.map((s) => s.key).join(",");
        if (keys != Object.keys(canvases).join(",")) {
            canvases = {};
            columns.replaceChildren(...series.map((s) => {
                const canvas = pieCanvas({ legend: false, onclick: (slice) => onselect(s.key, slice) });
                canvases[s.key] = canvas;
                return t.div(
                    { className: "band-chart" },
                    t.div({ className: "band-label" }, s.label),
                    t.div({ className: "chart-canvas-wrap" }, canvas),
                    t.div({ className: "chart-legend" }),
                );
            }));
        }

        series.forEach((s, i) => {
            canvases[s.key].update(s.slices);
            columns.children[i].querySelector(".chart-legend").replaceChildren(...s.slices.map((slice) => {
                return t.span(
                    { className: "legend-item" },
                    t.span({ className: "legend-swatch", style: `background:${slice.color}` }),
                    `${slice.label}: ${slice.value}`,
                );
            }));
        });
    };

    return el;
}
