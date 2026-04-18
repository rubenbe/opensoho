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

    let chartCanvas;
    let chartInst;
    let isLoading = false;
    let deviceIds = [];

    async function load() {
        isLoading = true;
        try {
            const records = await ApiClient.collection("connected_clients").getFullList({
                fields: "device,expand.device.name,expand.device.id",
                expand: "device",
                requestKey: "clients_per_device",
            });

            const counts = {};
            const idByName = {};
            for (const r of records) {
                const name = r.expand?.device?.name || r.device || "Unknown";
                const id = r.expand?.device?.id || r.device || "";
                counts[name] = (counts[name] || 0) + 1;
                idByName[name] = id;
            }

            const labels = Object.keys(counts);
            const data = labels.map((l) => counts[l]);
            const colors = labels.map((_, i) => PALETTE[i % PALETTE.length]);
            deviceIds = labels.map((l) => idByName[l]);

            if (chartInst) {
                chartInst.data.labels = labels;
                chartInst.data.datasets[0].data = data;
                chartInst.data.datasets[0].backgroundColor = colors;
                chartInst.update();
            }
        } catch (err) {
            if (!err?.isAbort) {
                ApiClient.error(err);
            }
        } finally {
            isLoading = false;
        }
    }

    onMount(() => {
        Chart.register(ArcElement, PieController, Tooltip, Legend);

        chartInst = new Chart(chartCanvas, {
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
                    const id = deviceIds[elements[0].index];
                    if (!id) return;
                    const filter = `device = "${id}"`;
                    push(`/collections?collection=connected_clients&filter=${encodeURIComponent(filter)}`);
                },
                plugins: {
                    legend: {
                        position: "bottom",
                        labels: { color: "#617079", boxWidth: 12, padding: 16 },
                    },
                    tooltip: {
                        callbacks: {
                            label: (ctx) => ` ${ctx.label}: ${ctx.parsed}`,
                        },
                    },
                },
            },
        });

        load();

        return () => chartInst?.destroy();
    });
</script>

<div class="chart-wrapper" class:loading={isLoading}>
    {#if isLoading}
        <div class="chart-loader loader" transition:scale={{ duration: 150 }} />
    {/if}
    <canvas bind:this={chartCanvas} class="chart-canvas" />
</div>

<style>
    .chart-wrapper {
        position: relative;
        width: 100%;
        height: 260px;
    }
    .chart-wrapper.loading .chart-canvas {
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
    .chart-canvas {
        cursor: pointer;
    }
</style>
