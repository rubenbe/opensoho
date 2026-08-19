<script>
    import { onMount } from "svelte";
    import { scale } from "svelte/transition";
    import ApiClient from "@/utils/ApiClient";
    import { Chart, ArcElement, PieController, Tooltip, Legend } from "chart.js";
    import { push } from "svelte-spa-router";

    const PALETTE = [
        "#4e79a7", "#f28e2b", "#e15759", "#76b7b2", "#59a14f",
        "#edc948", "#b07aa1", "#ff9da7", "#9c755f", "#bab0ac",
    ];

    let canvas24;
    let canvas5;
    let canvas6;
    let chart24;
    let chart5;
    let chart6;
    let isLoading = false;
    let channels24 = [];
    let channels5 = [];
    let channels6 = [];
    let show6 = false;
    let pendingData6 = null;

    function buildChartData(channelCounts) {
        const channels = Object.keys(channelCounts).map(Number).sort((a, b) => a - b);
        const labels = channels.map((c) => `Ch ${c}`);
        const data = channels.map((c) => channelCounts[c]);
        const colors = channels.map((_, i) => PALETTE[i % PALETTE.length]);
        return { channels, labels, data, colors };
    }

    function makeOnClick(band, getChannels) {
        return (_, elements) => {
            if (!elements.length) return;
            const ch = getChannels()[elements[0].index];
            if (ch == null) return;
            const filter = `channel = ${ch} && band = "${band}"`;
            push(`/collections?collection=connected_clients&filter=${encodeURIComponent(filter)}`);
        };
    }

    function makeChartOptions(band, getChannels) {
        return {
            responsive: true,
            maintainAspectRatio: false,
            onClick: makeOnClick(band, getChannels),
            plugins: {
                legend: {
                    position: "bottom",
                    labels: { color: "#617079", boxWidth: 12, padding: 12 },
                },
                tooltip: {
                    callbacks: {
                        label: (ctx) => ` ${ctx.label}: ${ctx.parsed}`,
                    },
                },
            },
        };
    }

    export async function load() {
        isLoading = true;
        try {
            const records = await ApiClient.collection("connected_clients").getFullList({
                fields: "channel,band",
                requestKey: "clients_per_channel",
            });

            const counts24 = {};
            const counts5 = {};
            const counts6 = {};
            for (const r of records) {
                if (!r.channel) continue;
                if (r.band === "2.4") {
                    counts24[r.channel] = (counts24[r.channel] || 0) + 1;
                } else if (r.band === "5") {
                    counts5[r.channel] = (counts5[r.channel] || 0) + 1;
                } else if (r.band === "6") {
                    counts6[r.channel] = (counts6[r.channel] || 0) + 1;
                }
            }

            const d24 = buildChartData(counts24);
            channels24 = d24.channels;
            if (chart24) {
                chart24.data.labels = d24.labels;
                chart24.data.datasets[0].data = d24.data;
                chart24.data.datasets[0].backgroundColor = d24.colors;
                chart24.update();
            }

            const d5 = buildChartData(counts5);
            channels5 = d5.channels;
            if (chart5) {
                chart5.data.labels = d5.labels;
                chart5.data.datasets[0].data = d5.data;
                chart5.data.datasets[0].backgroundColor = d5.colors;
                chart5.update();
            }

            const d6 = buildChartData(counts6);
            channels6 = d6.channels;
            show6 = channels6.length > 0;
            if (show6) {
                if (chart6) {
                    chart6.data.labels = d6.labels;
                    chart6.data.datasets[0].data = d6.data;
                    chart6.data.datasets[0].backgroundColor = d6.colors;
                    chart6.update();
                } else {
                    pendingData6 = d6;
                }
            }
        } catch (err) {
            if (!err?.isAbort) {
                ApiClient.error(err);
            }
        } finally {
            isLoading = false;
        }
    }

    function initChart(canvas, band, getChannels) {
        return new Chart(canvas, {
            type: "pie",
            data: {
                labels: [],
                datasets: [{ data: [], backgroundColor: [], borderWidth: 0 }],
            },
            options: makeChartOptions(band, getChannels),
        });
    }

    // canvas6 only exists in the DOM once show6 becomes true, so chart6 is
    // created lazily once both the flag and the canvas are available.
    $: if (show6 && canvas6 && !chart6) {
        chart6 = initChart(canvas6, "6", () => channels6);
        if (pendingData6) {
            chart6.data.labels = pendingData6.labels;
            chart6.data.datasets[0].data = pendingData6.data;
            chart6.data.datasets[0].backgroundColor = pendingData6.colors;
            chart6.update();
            pendingData6 = null;
        }
    }
    $: if (!show6 && chart6) {
        chart6.destroy();
        chart6 = null;
    }

    onMount(() => {
        Chart.register(ArcElement, PieController, Tooltip, Legend);
        chart24 = initChart(canvas24, "2.4", () => channels24);
        chart5 = initChart(canvas5, "5", () => channels5);

        load();

        return () => {
            chart24?.destroy();
            chart5?.destroy();
            chart6?.destroy();
        };
    });
</script>

<div class="channel-charts" class:loading={isLoading}>
    {#if isLoading}
        <div class="chart-loader loader" transition:scale={{ duration: 150 }} />
    {/if}
    <div class="band-chart">
        <div class="band-label">2.4 GHz</div>
        <canvas bind:this={canvas24} class="chart-canvas" />
    </div>
    <div class="band-chart">
        <div class="band-label">5 GHz</div>
        <canvas bind:this={canvas5} class="chart-canvas" />
    </div>
    {#if show6}
        <div class="band-chart">
            <div class="band-label">6 GHz</div>
            <canvas bind:this={canvas6} class="chart-canvas" />
        </div>
    {/if}
</div>

<style>
    .channel-charts {
        position: relative;
        display: flex;
        gap: var(--baseSpacing);
        width: 100%;
        height: 260px;
    }
    .channel-charts.loading .chart-canvas {
        opacity: 0.5;
        pointer-events: none;
    }
    .chart-loader {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        z-index: 1;
    }
    .band-chart {
        flex: 1;
        display: flex;
        flex-direction: column;
        min-width: 0;
    }
    .band-label {
        text-align: center;
        font-size: var(--smFontSize);
        font-weight: 600;
        color: var(--txtHintColor);
        margin-bottom: 4px;
    }
    .chart-canvas {
        cursor: pointer;
        flex: 1;
        min-height: 0;
    }
</style>
