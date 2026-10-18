import { PALETTE, bandPieCharts, loading, openRecords, pieCanvas, pieChart } from "./charts.js";

const HEALTH_FILTERS = [
    `health_status = "healthy"`,
    `health_status = "unhealthy"`,
    `health_status = ""`,
];

export function deviceHealthChart() {
    const state = store({ isLoading: false });

    const canvas = pieCanvas({
        onclick: (slice) => openRecords("devices", slice.filter),
    });

    async function load() {
        await loading(state, async () => {
            const records = await app.pb.collection("devices").getFullList({
                fields: "health_status",
                requestKey: "device_health_stats",
            });

            const counts = { healthy: 0, unhealthy: 0, unknown: 0 };
            for (const r of records) {
                const s = r.health_status || "unknown";
                counts[s] = (counts[s] || 0) + 1;
            }

            canvas.update([
                { label: "Healthy", value: counts.healthy, color: "#32ad84", filter: HEALTH_FILTERS[0] },
                { label: "Unhealthy", value: counts.unhealthy, color: "#e34562", filter: HEALTH_FILTERS[1] },
                { label: "Unknown", value: counts.unknown, color: "#a5b0c0", filter: HEALTH_FILTERS[2] },
            ]);
        });
    }

    return { el: pieChart(state, canvas), load };
}

export function clientsPerDeviceChart() {
    const state = store({ isLoading: false });

    const canvas = pieCanvas({
        onclick: (slice) => {
            if (slice.deviceId) {
                openRecords("connected_clients", `device = "${slice.deviceId}"`);
            }
        },
    });

    async function load() {
        await loading(state, async () => {
            const records = await app.pb.collection("connected_clients").getFullList({
                fields: "device,expand.device.name,expand.device.id",
                expand: "device",
                requestKey: "clients_per_device",
            });

            const counts = {};
            const idByName = {};
            for (const r of records) {
                const name = r.expand?.device?.name || r.device || "Unknown";
                counts[name] = (counts[name] || 0) + 1;
                idByName[name] = r.expand?.device?.id || r.device || "";
            }

            canvas.update(Object.keys(counts).map((name, i) => ({
                label: name,
                value: counts[name],
                color: PALETTE[i % PALETTE.length],
                deviceId: idByName[name],
            })));
        });
    }

    return { el: pieChart(state, canvas), load };
}

const BANDS = [
    { key: "2.4", label: "2.4 GHz" },
    { key: "5", label: "5 GHz" },
    { key: "6", label: "6 GHz" },
];

export function clientsPerChannelChart() {
    const state = store({ isLoading: false });

    const el = bandPieCharts(state, (band, slice) => {
        openRecords("connected_clients", `channel = ${slice.channel} && band = "${band}"`);
    });

    async function load() {
        await loading(state, async () => {
            const records = await app.pb.collection("connected_clients").getFullList({
                fields: "channel,band",
                requestKey: "clients_per_channel",
            });

            const counts = { "2.4": {}, "5": {}, "6": {} };
            for (const r of records) {
                if (!r.channel || !counts[r.band]) continue;
                counts[r.band][r.channel] = (counts[r.band][r.channel] || 0) + 1;
            }

            el.update(
                BANDS
                    // 2.4 and 5 GHz always render; 6 GHz only when it has clients.
                    .filter((b) => b.key !== "6" || Object.keys(counts["6"]).length > 0)
                    .map((b) => {
                        const channels = Object.keys(counts[b.key]).map(Number).sort((x, y) => x - y);
                        return {
                            key: b.key,
                            label: b.label,
                            slices: channels.map((c, i) => ({
                                label: `Ch ${c}`,
                                value: counts[b.key][c],
                                color: PALETTE[i % PALETTE.length],
                                channel: c,
                            })),
                        };
                    }),
            );
        });
    }

    return { el, load };
}

// Purely presentational: the band/generation categorization itself comes
// from the server (apiWifiVersions in opensoho.go), which resolves it the
// same way a radio is actually configured (resolveRadioBand,
// htModeGenerationPrefix) instead of re-deriving it here.
const WIFI_VERSION_COLORS = {
    "EHT": "#4e79a7",
    "HE": "#59a14f",
    "VHT": "#edc948",
    "HT": "#f28e2b",
    "": "#a5b0c0",
};

export function wifiVersionChart() {
    const state = store({ isLoading: false });

    const el = bandPieCharts(state, (_, slice) => openRecords("radios", slice.filter));

    async function load() {
        await loading(state, async () => {
            const res = await app.pb.send("/api/v1/wifi-versions", {
                method: "GET",
                requestKey: "wifi_versions",
            });

            el.update((res.bands || []).map((b) => ({
                key: b.key,
                label: b.label,
                slices: b.slices.map((s) => ({
                    label: s.label,
                    value: s.count,
                    color: WIFI_VERSION_COLORS[s.prefix] ?? WIFI_VERSION_COLORS[""],
                    filter: s.filter,
                })),
            })));
        });
    }

    return { el, load };
}
