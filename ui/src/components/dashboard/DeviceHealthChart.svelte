<script>
    import { onMount } from "svelte";
    import { scale } from "svelte/transition";
    import ApiClient from "@/utils/ApiClient";
    import { Chart, ArcElement, PieController, Tooltip, Legend } from "chart.js";

    let chartCanvas;
    let chartInst;
    let isLoading = false;

    async function load() {
        isLoading = true;
        try {
            const records = await ApiClient.collection("devices").getFullList({
                fields: "health_status",
                requestKey: "device_health_stats",
            });

            const counts = { healthy: 0, unhealthy: 0, unknown: 0 };
            for (const r of records) {
                const s = r.health_status || "unknown";
                counts[s] = (counts[s] || 0) + 1;
            }

            if (chartInst) {
                chartInst.data.datasets[0].data = [counts.healthy, counts.unhealthy, counts.unknown];
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
                labels: ["Healthy", "Unhealthy", "Unknown"],
                datasets: [
                    {
                        data: [0, 0, 0],
                        backgroundColor: ["#32ad84", "#e34562", "#a5b0c0"],
                        borderWidth: 0,
                    },
                ],
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: "bottom",
                        labels: {
                            color: "#617079",
                            boxWidth: 12,
                            padding: 16,
                        },
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
</style>
