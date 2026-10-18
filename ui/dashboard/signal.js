import { bandPieCharts, loader, loading, openRecords, pieCanvas, pieChart } from "./charts.js";

// Purely presentational: the tiering, the per-band split, the bins and the
// record filters all come from the server (apiClientSignalQuality in opensoho.go).
const COLORS = {
    excellent: "#32ad84",
    good: "#59a14f",
    fair: "#f28e2b",
    poor: "#e05c30",
    critical: "#e34562",
};

// Red -> amber -> green, matching the danger / warning / success colors.
// Hardcoded so the badges and the track stay in sync in both color schemes.
const STOPS = [
    { at: 0, color: [227, 69, 98] },
    { at: 0.5, color: [255, 148, 77] },
    { at: 1, color: [50, 173, 132] },
];

function gradientColor(v) {
    v = Math.max(0, Math.min(1, v));
    let a = STOPS[0];
    let b = STOPS[STOPS.length - 1];
    for (let i = 0; i < STOPS.length - 1; i++) {
        if (v >= STOPS[i].at && v <= STOPS[i + 1].at) {
            a = STOPS[i];
            b = STOPS[i + 1];
            break;
        }
    }
    const span = b.at - a.at || 1;
    const local = (v - a.at) / span;
    const rgb = a.color.map((c, i) => Math.round(c + (b.color[i] - c) * local));
    return `rgb(${rgb.join(",")})`;
}

const TRACK_GRADIENT = `linear-gradient(90deg, ${
    STOPS.map((s) => `${gradientColor(s.at)} ${s.at * 100}%`).join(", ")
})`;

/**
 * Signal strength bins drawn as badges on a linear dBm scale.
 *
 * @param  {Array}    series   [{ key, label, bins: [{ min, max, label, count, filter }] }]
 * @param  {Object}   scale    { min, max } in dBm
 * @param  {boolean}  showLabels
 * @param  {Function} onselect Called with the clicked bin.
 * @return {Element}
 */
function signalScaleBars(series, scale, showLabels, onselect) {
    const fraction = (v) => (v - scale.min) / (scale.max - scale.min);

    const ticks = [];
    for (let v = scale.min; v <= scale.max; v += 10) {
        ticks.push(v);
    }

    return t.div(
        { className: "scale-bars" },
        series.map((band) => {
            return t.div(
                { className: `scale-row ${showLabels ? "labeled" : ""}` },
                showLabels ? t.div({ className: "band-label" }, band.label) : null,
                t.div(
                    { className: "scale-main" },
                    t.div(
                        { className: "scale-plot" },
                        band.bins.filter((bin) => bin.count > 0).map((bin) => {
                            const center = fraction((bin.min + bin.max) / 2);
                            return t.button(
                                {
                                    type: "button",
                                    className: "scale-badge",
                                    style: `left: ${center * 100}%; background: ${gradientColor(center)}`,
                                    ariaLabel: app.attrs.tooltip(`${bin.count} clients, ${bin.label}`, "top"),
                                    onclick: () => onselect(bin),
                                },
                                bin.count,
                            );
                        }),
                        t.div({ className: "scale-track", style: `background: ${TRACK_GRADIENT}` }),
                    ),
                    t.div(
                        { className: "scale-axis" },
                        ticks.map((v) => t.span({ className: "tick", style: `left: ${fraction(v) * 100}%` }, v)),
                    ),
                ),
            );
        }),
    );
}

function toSeries(band) {
    return {
        key: band.key,
        label: band.label,
        slices: band.slices.map((s) => ({
            label: s.label,
            value: s.count,
            color: COLORS[s.tier],
            filter: s.filter,
        })),
        bins: band.bins || [],
    };
}

/**
 * @param  {Object} prefs Store with the linearScale and splitByBand view options.
 * @return {{el: Element, load: Function}}
 */
export function clientSignalQualityChart(prefs) {
    const state = store({ isLoading: false });

    let result = { overall: { slices: [], bins: [] }, bands: [], scale: { min: -90, max: -30 } };

    const openClients = (filter) => openRecords("connected_clients", filter);

    const overallPie = pieCanvas({ onclick: (slice) => openClients(slice.filter) });
    const overallChart = pieChart(state, overallPie);
    const bandPies = bandPieCharts(state, (_, slice) => openClients(slice.filter));

    let watcher;

    const body = t.div({
        className: "signal-quality",
        onmount: () => {
            watcher = watch(() => [prefs.linearScale, prefs.splitByBand], render);
        },
        onunmount: () => watcher?.unwatch(),
    });

    // replaces the card content, unless it is already shown (which would
    // needlessly destroy and recreate its chart)
    function show(el) {
        if (body.firstChild !== el || body.childNodes.length > 1) {
            body.replaceChildren(el);
        }
    }

    function render() {
        if (prefs.linearScale) {
            const series = prefs.splitByBand
                ? result.bands
                : [{ key: "all", label: "All", bins: result.overall.bins }];
            show(t.div(
                { className: () => `scale-bars-wrapper ${state.isLoading ? "loading" : ""}` },
                loader(state),
                signalScaleBars(series, result.scale, prefs.splitByBand, (bin) => openClients(bin.filter)),
            ));
        } else if (prefs.splitByBand) {
            bandPies.update(result.bands);
            show(bandPies);
        } else {
            overallPie.update(result.overall.slices);
            show(overallChart);
        }
    }

    async function load() {
        await loading(state, async () => {
            const res = await app.pb.send("/api/v1/client-signal-quality", {
                method: "GET",
                requestKey: "clients_signal_quality",
            });

            result = {
                overall: toSeries(res.overall),
                bands: (res.bands || []).map(toSeries),
                scale: res.scale || result.scale,
            };
        });

        // after loading, so that newly shown elements don't mount mid-load
        render();
    }

    return { el: body, load };
}
