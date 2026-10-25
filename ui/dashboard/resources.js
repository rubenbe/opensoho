import { openRecords } from "./charts.js";
import { formatTime, historyChart } from "./history.js";
import { healthyDeviceIds } from "./selection.js";
import { byteStep, formatBytes, formatNumber } from "./timeseries.js";

// Metrics of one unit share a chart, so a chart never needs a second y axis.
const UNITS = [
    { key: "load", label: "Load", format: formatNumber },
    { key: "bytes", label: "Memory", format: formatBytes, step: byteStep },
    { key: "count", label: "CPUs", format: formatNumber },
];

// The columns of /api/v1/device-resources (apiDeviceResources in opensoho.go).
const METRICS = [
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

const EVENT_LABELS = { reboot: "rebooted" };

/**
 * Resource history card: the selected columns of the device_resources
 * collection over time, for the selected devices.
 *
 * @return {{el: Element, load: Function}}
 */
export function resourceHistoryChart() {
    return historyChart({
        storageKey: "dashboard_resource_history",
        names: { item: "Device", items: "Devices", none: "No devices" },
        units: UNITS,
        metrics: METRICS,
        defaultMetrics: ["load1"],
        loadItems: () => app.pb.collection("devices").getFullList({
            fields: "id,name,health_status",
            sort: "name",
            requestKey: "resource_history_devices",
        }),
        itemName: (device) => device.name || device.id,
        defaultItems: (devices) => devices.map((d) => d.id),
        selectActions: (devices) => [
            { label: "All devices", ids: devices.map((d) => d.id) },
            { label: "All healthy devices", ids: healthyDeviceIds(devices) },
        ],
        loadHistory: (id, { from, to }) => app.pb.send(`/api/v1/device-resources/${encodeURIComponent(id)}`, {
            method: "GET",
            query: { from, to },
            requestKey: `resource_history_${id}`,
        }),
        openItem: (device) => openRecords("devices", `id = "${device.id}"`),
        openTitle: "Show this device",
        eventLabel: (mark, name) => `${name} ${EVENT_LABELS[mark.type] || mark.type} at ${formatTime(mark.x, true)}`,
        eventsTitle: "Reboots",
        emptyMessage: "No samples in this period. History is only kept when the history_retention_days setting is above 0, which it is not by default.",
    });
}
