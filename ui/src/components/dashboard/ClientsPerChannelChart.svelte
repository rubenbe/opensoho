<script>
    import { onMount } from "svelte";
    import { scale } from "svelte/transition";
    import ApiClient from "@/utils/ApiClient";
    import { Chart, ArcElement, PieController, Tooltip, Legend } from "chart.js";

    const PALETTE = [
        "#4e79a7", "#f28e2b", "#e15759", "#76b7b2", "#59a14f",
        "#edc948", "#b07aa1", "#ff9da7", "#9c755f", "#bab0ac",
    ];

    let canvas24;
    let canvas5;
    let chart24;
    let chart5;
    let isLoading = false;

    function buildChartData(channelCounts) {
        const channels = Object.keys(channelCounts).map(Number).sort((a, b) => a - b);
        const labels = channels.map((c) => `Ch ${c}`);
        const data = channels.map((c) => channelCounts[c]);
        const colors = channels.map((_, i) => PALETTE[i % PALETTE.length]);
        return { labels, data, colors };
    }

    function makeChartOptions() {
        return {
            responsive: true,
            maintainAspectRatio: false,
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

    async function load() {
        isLoading = true;
        try {
            const records = await ApiClient.collection("connected_clients").getFullList({
                fields: "channel,band",
                requestKey: "clients_per_channel",
            });

            const counts24 = {};
            const counts5 = {};
            for (const r of records) {
                if (!r.channel) continue;
                if (r.band === "2.4") {
                    counts24[r.channel] = (counts24[r.channel] || 0) + 1;
                } else if (r.band === "5") {
                    counts5[r.channel] = (counts5[r.channel] || 0) + 1;
                }
            }

            const d24 = buildChartData(counts24);
            if (chart24) {
                chart24.data.labels = d24.labels;
                chart24.data.datasets[0].data = d24.data;
                chart24.data.datasets[0].backgroundColor = d24.colors;
                chart24.update();
            }

            const d5 = buildChartData(counts5);
            if (chart5) {
                chart5.data.labels = d5.labels;
                chart5.data.datasets[0].data = d5.data;
                chart5.data.datasets[0].backgroundColor = d5.colors;
                chart5.update();
            }
        } catch (err) {
            if (!err?.isAbort) {
                ApiClient.error(err);
            }
        } finally {
            isLoading = false;
        }
    }

    function initChart(canvas) {
        return new Chart(canvas, {
            type: "pie",
            data: {
                labels: [],
                datasets: [{ data: [], backgroundColor: [], borderWidth: 0 }],
            },
            options: makeChartOptions(),
        });
    }

    onMount(() => {
        Chart.register(ArcElement, PieController, Tooltip, Legend);
        chart24 = initChart(canvas24);
        chart5 = initChart(canvas5);

        load();

        return () => {
            chart24?.destroy();
            chart5?.destroy();
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
        flex: 1;
        min-height: 0;
    }
</style>
