import { loadChartJs } from "./charts.js";
import { frequencyOverview } from "./frequency.js";
import { networkOverview } from "./network.js";
import { clientsPerChannelChart, clientsPerDeviceChart, deviceHealthChart, wifiVersionChart } from "./pies.js";
import { clientSignalQualityChart } from "./signal.js";

const SIGNAL_SPLIT_STORAGE_KEY = "dashboard_signal_split_by_band";
const SIGNAL_LINEAR_STORAGE_KEY = "dashboard_signal_linear_scale";

const REFRESH_INTERVAL = 15000;

// Remember dashboard card view preferences (e.g. the "Client Signal Quality"
// menu options) across page loads.
function loadPref(key) {
    try {
        return window.localStorage?.getItem(key) === "1";
    } catch (_) {
        return false;
    }
}

function savePref(key, value) {
    try {
        window.localStorage?.setItem(key, value ? "1" : "0");
    } catch (_) {}
}

function card(title, body, { wide = false, menu = null } = {}) {
    return t.div(
        { className: `dashboard-card ${wide ? "wide" : ""}` },
        t.div({ className: "card-header" }, t.h6({ className: "card-title" }, title), menu),
        t.div({ className: "card-body" }, body),
    );
}

function cardMenu(label, ...items) {
    const dropdown = t.div({ className: "dropdown sm nowrap gap-0", popover: "auto" }, ...items);

    return t.button(
        {
            type: "button",
            ariaLabel: label,
            title: label,
            className: "btn sm secondary transparent circle",
            popoverTargetElement: dropdown,
        },
        t.i({ className: "ri-more-line", ariaHidden: true }),
        dropdown,
    );
}

function toggle(label, getter, setter) {
    const id = "dashboard_toggle_" + app.utils.randomString();

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
                checked: getter,
                onchange: (e) => setter(e.target.checked),
            }),
            t.label({ htmlFor: id }, label),
        ),
    );
}

export async function pageDashboard() {
    app.store.title = "Dashboard";

    await loadChartJs();

    const signalPrefs = store({
        splitByBand: loadPref(SIGNAL_SPLIT_STORAGE_KEY),
        linearScale: loadPref(SIGNAL_LINEAR_STORAGE_KEY),
    });

    const deviceHealth = deviceHealthChart();
    const clientsPerDevice = clientsPerDeviceChart();
    const clientsPerChannel = clientsPerChannelChart();
    const wifiVersions = wifiVersionChart();
    const signalQuality = clientSignalQualityChart(signalPrefs);
    const frequencies = frequencyOverview();
    const network = networkOverview();

    const cards = [
        deviceHealth,
        clientsPerDevice,
        clientsPerChannel,
        wifiVersions,
        signalQuality,
        frequencies,
        network,
    ];

    function refreshAll() {
        cards.forEach((c) => c.load());
    }

    let refreshInterval;

    return t.div(
        {
            pbEvent: "pageDashboard",
            className: "page page-dashboard",
            onmount: () => {
                refreshAll();
                refreshInterval = setInterval(refreshAll, REFRESH_INTERVAL);
            },
            onunmount: () => clearInterval(refreshInterval),
        },
        t.div(
            { className: "page-content full-height" },
            t.header(
                { className: "page-header" },
                t.nav({ className: "breadcrumbs" }, t.div(null, "Dashboard")),
                t.div(
                    { className: "inline-flex gap-sm" },
                    app.components.refreshButton({ onclick: refreshAll }),
                ),
            ),
            t.div(
                { className: "dashboard-grid" },
                card("Device Health", deviceHealth.el),
                card("Clients per Device", clientsPerDevice.el),
                card("Clients per Channel", clientsPerChannel.el),
                card("AP WiFi Versions", wifiVersions.el),
                card("Client Signal Quality", signalQuality.el, {
                    menu: cardMenu(
                        "Client signal quality options",
                        toggle("Linear signal scale", () => signalPrefs.linearScale, (v) => {
                            signalPrefs.linearScale = v;
                            savePref(SIGNAL_LINEAR_STORAGE_KEY, v);
                        }),
                        toggle("Split per frequency band", () => signalPrefs.splitByBand, (v) => {
                            signalPrefs.splitByBand = v;
                            savePref(SIGNAL_SPLIT_STORAGE_KEY, v);
                        }),
                    ),
                }),
                card("Frequency Overview", frequencies.el, { wide: true }),
                card("Network Overview", network.el, { wide: true }),
            ),
            t.footer({ className: "page-footer" }, app.components.credits()),
        ),
    );
}
