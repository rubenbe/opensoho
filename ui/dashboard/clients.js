import { openRecords } from "./charts.js";
import { formatTime, historyChart, MAX_ITEMS } from "./history.js";
import { topByTraffic } from "./selection.js";
import { bandLabel, bitRateStep, clientLocation, formatBitRate, formatDbm } from "./timeseries.js";

const UNITS = [
    { key: "rate", label: "Traffic", format: formatBitRate, step: bitRateStep },
    // negative, so the axis follows the values instead of starting at 0
    { key: "dbm", label: "Signal", format: formatDbm, beginAtZero: false },
];

// The values of /api/v1/client-resources (apiClientResources in opensoho.go):
// the traffic in bytes per second, from the client's point of view, and the
// average signal (RSSI).
const METRICS = [
    { key: "download", label: "Download", unit: "rate" },
    { key: "upload", label: "Upload", unit: "rate" },
    { key: "signal", label: "Signal", unit: "dbm" },
];

function eventLabel(mark, name) {
    const at = formatTime(mark.x, true);
    const ap = mark.event.device_name || mark.event.device;
    const band = bandLabel(mark.event.band);
    const onBand = band ? ` (${band})` : "";
    switch (mark.type) {
        case "roam":
            return `${name} roamed to ${ap}${onBand} at ${at}`;
        case "ssid":
            return `${name} switched to SSID ${mark.event.ssid}${onBand} at ${at}`;
        case "band":
            return `${name} switched to ${band} on ${ap} at ${at}`;
        case "reconnect":
            return `${name} reconnected to ${ap}${onBand} at ${at}`;
        case "disconnect":
            return `${name} disconnected from ${ap}${onBand} at ${at}`;
        case "reboot":
            return `${ap}, used by ${name}, rebooted at ${at}`;
        default:
            return `${name}: ${mark.type} at ${at}`;
    }
}

/**
 * Client history card: the traffic of the selected clients over time.
 *
 * @return {{el: Element, load: Function}}
 */
export function clientHistoryChart() {
    return historyChart({
        storageKey: "dashboard_client_history",
        names: { item: "Client", items: "Clients", none: "No clients with history in this period" },
        units: UNITS,
        metrics: METRICS,
        defaultMetrics: ["download", "upload"],
        // only the clients with history in the period, by name
        loadItems: async ({ from, to }) => {
            const res = await app.pb.send("/api/v1/client-traffic", {
                method: "GET",
                query: { from, to },
                requestKey: "client_history_clients",
            });
            return res.clients || [];
        },
        itemName: (client) => client.name,
        // the AP and band of its last sample in the period
        itemDetail: clientLocation,
        defaultItems: (clients) => topByTraffic(clients, MAX_ITEMS),
        selectActions: (clients) => [
            { label: `Top ${MAX_ITEMS} by traffic`, ids: topByTraffic(clients, MAX_ITEMS) },
        ],
        loadHistory: (id, { from, to }) => app.pb.send(`/api/v1/client-resources/${encodeURIComponent(id)}`, {
            method: "GET",
            query: { from, to },
            requestKey: `client_history_${id}`,
        }),
        openItem: (client) => openRecords("clients", `id = "${client.id}"`),
        openTitle: "Show this client",
        eventLabel,
        eventsTitle: "Events",
        emptyMessage: "No samples in this period. History is only kept when the history_retention_days setting is above 0, which it is not by default.",
    });
}
