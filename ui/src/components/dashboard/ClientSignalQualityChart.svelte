<script>
    import { onMount } from "svelte";
    import { scale } from "svelte/transition";
    import ApiClient from "@/utils/ApiClient";
    import { Chart, ArcElement, PieController, Tooltip, Legend } from "chart.js";
    import { push } from "svelte-spa-router";
    import BandPieCharts from "@/components/dashboard/BandPieCharts.svelte";

    // Purely presentational: the tiering, the per-band split and the record
    // filters all come from the server (apiClientSignalQuality in opensoho.go).
    const COLORS = {
        excellent: "#32ad84",
        good: "#59a14f",
        fair: "#f28e2b",
        poor: "#e05c30",
        critical: "#e34562",
    };

    export let splitByBand = false;

    Chart.register(ArcElement, PieController, Tooltip, Legend);

    let isLoading = false;
    let overall = [];
    let series = [];

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
        };
    }

    export async function load() {
        isLoading = true;
        try {
            const res = await ApiClient.send("/api/v1/client-signal-quality", {
                method: "GET",
                requestKey: "clients_signal_quality",
            });

            overall = toSeries(res.overall).slices;
            series = (res.bands || []).map(toSeries);
        } catch (err) {
            if (!err?.isAbort) {
                ApiClient.error(err);
            }
        } finally {
            isLoading = false;
        }
    }

    function openClients(filter) {
        push(`/collections?collection=connected_clients&filter=${encodeURIComponent(filter)}`);
    }

    function onBandSelect(e) {
        openClients(e.detail.slice.filter);
    }

    function pie(canvas, slices) {
        let current = slices;
        const chart = new Chart(canvas, {
            type: "pie",
            data: {
                labels: slices.map((sl) => sl.label),
                datasets: [
                    {
                        data: slices.map((sl) => sl.value),
                        backgroundColor: slices.map((sl) => sl.color),
                        borderWidth: 0,
                    },
                ],
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                onClick: (_, elements) => {
                    if (!elements.length) return;
                    openClients(current[elements[0].index].filter);
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

        return {
            update(next) {
                current = next;
                chart.data.labels = next.map((sl) => sl.label);
                chart.data.datasets[0].data = next.map((sl) => sl.value);
                chart.data.datasets[0].backgroundColor = next.map((sl) => sl.color);
                chart.update();
            },
            destroy() {
                chart.destroy();
            },
        };
    }

    onMount(() => {
        load();
    });
</script>

{#if splitByBand}
    <BandPieCharts {series} {isLoading} on:select={onBandSelect} />
{:else}
    <div class="chart-wrapper" class:loading={isLoading}>
        {#if isLoading}
            <div class="chart-loader loader" transition:scale={{ duration: 150 }} />
        {/if}
        <canvas use:pie={overall} class="chart-canvas" />
    </div>
{/if}

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
