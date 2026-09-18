<script>
    import { createEventDispatcher } from "svelte";
    import { scale } from "svelte/transition";
    import { Chart, ArcElement, PieController, Tooltip, Legend } from "chart.js";

    export let series = []; // [{ key, label, slices: [{ label, value, color }] }]
    export let isLoading = false;
    export let height = 180; // px, fixed canvas height so all columns line up

    const dispatch = createEventDispatcher();

    Chart.register(ArcElement, PieController, Tooltip, Legend);

    function pieChart(canvas, s) {
        const chart = new Chart(canvas, {
            type: "pie",
            data: {
                labels: s.slices.map((sl) => sl.label),
                datasets: [
                    {
                        data: s.slices.map((sl) => sl.value),
                        backgroundColor: s.slices.map((sl) => sl.color),
                        borderWidth: 0,
                    },
                ],
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                onClick: (_, elements) => {
                    if (!elements.length) return;
                    const slice = current.slices[elements[0].index];
                    if (!slice) return;
                    dispatch("select", { band: current.key, slice });
                },
                plugins: {
                    // Drawn with the built-in legend disabled so the pie always
                    // fills the whole, fixed-height canvas and lines up across
                    // columns. The legend is instead rendered as plain HTML below.
                    legend: { display: false },
                    tooltip: {
                        callbacks: {
                            label: (ctx) => ` ${ctx.label}: ${ctx.parsed}`,
                        },
                    },
                },
            },
        });

        let current = s;

        return {
            update(next) {
                current = next;
                chart.data.labels = next.slices.map((sl) => sl.label);
                chart.data.datasets[0].data = next.slices.map((sl) => sl.value);
                chart.data.datasets[0].backgroundColor = next.slices.map((sl) => sl.color);
                chart.update();
            },
            destroy() {
                chart.destroy();
            },
        };
    }
</script>

<div class="band-charts" class:loading={isLoading}>
    {#if isLoading}
        <div class="chart-loader loader" transition:scale={{ duration: 150 }} />
    {/if}
    {#each series as s (s.key)}
        <div class="band-chart">
            <div class="band-label">{s.label}</div>
            <div class="chart-canvas-wrap" style="height: {height}px">
                <canvas class="chart-canvas" use:pieChart={s} />
            </div>
            <div class="chart-legend">
                {#each s.slices as slice}
                    <span class="legend-item">
                        <span class="legend-swatch" style="background:{slice.color}" />
                        {slice.label}: {slice.value}
                    </span>
                {/each}
            </div>
        </div>
    {/each}
</div>

<style>
    .band-charts {
        position: relative;
        display: flex;
        align-items: flex-start;
        gap: var(--baseSpacing);
        width: 100%;
        min-height: 260px;
    }
    .band-charts.loading .chart-canvas {
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
    /* Fixed height (rather than flex:1) so every pie renders at the same
       size and sits flush against the top of its column, regardless of
       how many legend rows wrap below it. */
    .chart-canvas-wrap {
        position: relative;
        width: 100%;
    }
    .chart-canvas {
        cursor: pointer;
    }
    .chart-legend {
        display: flex;
        flex-wrap: wrap;
        justify-content: center;
        gap: 4px 12px;
        margin-top: 8px;
        font-size: var(--smFontSize);
        color: var(--txtHintColor);
    }
    .legend-item {
        display: inline-flex;
        align-items: center;
        gap: 4px;
    }
    .legend-swatch {
        width: 10px;
        height: 10px;
        border-radius: 2px;
        flex-shrink: 0;
    }
</style>
