<script>
    import { onMount } from "svelte";
    import { scale } from "svelte/transition";
    import ApiClient from "@/utils/ApiClient";
    import { Chart, ArcElement, PieController, Tooltip, Legend } from "chart.js";
    import { push } from "svelte-spa-router";

    const TIERS = [
        { label: "Excellent", min: -50,  max: null, color: "#32ad84", filter: `signal >= -50` },
        { label: "Good",      min: -70,  max: -50,  color: "#59a14f", filter: `signal >= -70 && signal < -50` },
        { label: "Fair",      min: -78,  max: -70,  color: "#f28e2b", filter: `signal >= -78 && signal < -70` },
        { label: "Poor",      min: -85,  max: -78,  color: "#e05c30", filter: `signal >= -85 && signal < -79` },
        { label: "Critical",  min: null, max: -85,  color: "#e34562", filter: `signal < -85` },
    ];

    function classify(signal) {
        if (signal >= -50) return 0;
        if (signal >= -70) return 1;
        if (signal >= -78) return 2;
        if (signal >= -85) return 3;
        return 4;
    }

    let chartCanvas;
    let chartInst;
    let isLoading = false;

    async function load() {
        isLoading = true;
        try {
            const records = await ApiClient.collection("connected_clients").getFullList({
                fields: "signal",
                requestKey: "clients_signal_quality",
            });

            const counts = [0, 0, 0, 0, 0];
            for (const r of records) {
                counts[classify(r.signal)]++;
            }

            if (chartInst) {
                chartInst.data.datasets[0].data = counts;
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
                labels: TIERS.map((t) => t.label),
                datasets: [
                    {
                        data: [0, 0, 0, 0, 0],
                        backgroundColor: TIERS.map((t) => t.color),
                        borderWidth: 0,
                    },
                ],
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                onClick: (_, elements) => {
                    if (!elements.length) return;
                    const filter = TIERS[elements[0].index].filter;
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
