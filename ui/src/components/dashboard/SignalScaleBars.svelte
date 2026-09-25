<script>
    import { createEventDispatcher } from "svelte";
    import { scale as scaleTransition } from "svelte/transition";
    import tooltip from "@/actions/tooltip";

    // Purely presentational, like BandPieCharts: the bins, their counts and their
    // record filters all come from the server (apiClientSignalQuality in
    // opensoho.go). series shape: [{ key, label, bins: [{ min, max, label, count,
    // filter }] }].
    export let series = [];
    export let isLoading = false;
    export let scale = { min: -90, max: -30 };
    export let showLabels = false;

    const dispatch = createEventDispatcher();

    // Red -> amber -> green, matching --dangerColor / --warningColor /
    // --successColor. Those three are NOT overridden in dark mode (only their
    // pale *AltColor companions are), so a gradient built from them looks the
    // same in both themes; hardcoding the stops here keeps the badges and the
    // track in exact sync.
    const STOPS = [
        { at: 0, color: [227, 69, 98] }, // --dangerColor
        { at: 0.5, color: [255, 148, 77] }, // --warningColor
        { at: 1, color: [50, 173, 132] }, // --successColor
    ];

    function gradientColor(t) {
        t = Math.max(0, Math.min(1, t));
        let a = STOPS[0];
        let b = STOPS[STOPS.length - 1];
        for (let i = 0; i < STOPS.length - 1; i++) {
            if (t >= STOPS[i].at && t <= STOPS[i + 1].at) {
                a = STOPS[i];
                b = STOPS[i + 1];
                break;
            }
        }
        const span = b.at - a.at || 1;
        const local = (t - a.at) / span;
        const rgb = a.color.map((c, i) => Math.round(c + (b.color[i] - c) * local));
        return `rgb(${rgb.join(",")})`;
    }

    const trackGradient = `linear-gradient(90deg, ${STOPS.map((s) => `${gradientColor(s.at)} ${s.at * 100}%`).join(", ")})`;

    function fraction(v) {
        return (v - scale.min) / (scale.max - scale.min);
    }

    function position(bin) {
        return fraction((bin.min + bin.max) / 2) * 100;
    }

    function ticks() {
        const out = [];
        for (let v = scale.min; v <= scale.max; v += 10) {
            out.push(v);
        }
        return out;
    }

    function onSelect(band, bin) {
        dispatch("select", { band: band.key, bin });
    }
</script>

<div class="scale-bars" class:loading={isLoading}>
    {#if isLoading}
        <div class="scale-loader loader" transition:scaleTransition={{ duration: 150 }} />
    {/if}
    {#each series as band (band.key)}
        <div class="scale-row" class:labeled={showLabels}>
            {#if showLabels}
                <div class="band-label">{band.label}</div>
            {/if}
            <div class="scale-main">
            <div class="scale-plot">
                {#each band.bins.filter((b) => b.count > 0) as bin (bin.min)}
                    <button
                        type="button"
                        class="badge"
                        style="left: {position(bin)}%; background: {gradientColor(fraction((bin.min + bin.max) / 2))}"
                        aria-label="{bin.count} client{bin.count === 1 ? '' : 's'} {bin.label}"
                        use:tooltip={{ text: `${bin.count} clients, ${bin.label}`, position: "top" }}
                        on:click={() => onSelect(band, bin)}
                    >
                        {bin.count}
                    </button>
                {/each}
                <div class="scale-track" style="background: {trackGradient}" />
            </div>
            <div class="scale-axis">
                {#each ticks() as t (t)}
                    <span class="tick" style="left: {fraction(t) * 100}%">{t}</span>
                {/each}
            </div>
            </div>
        </div>
    {/each}
</div>

<style>
    .scale-bars {
        position: relative;
        width: 100%;
        min-height: 100px;
    }
    .scale-bars.loading {
        opacity: 0.5;
        pointer-events: none;
    }
    .scale-loader {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        z-index: 1;
    }
    .scale-row {
        margin-bottom: 22px;
    }
    /* Label beside the scale (not above it) so three bands fit the card. */
    .scale-row.labeled {
        display: flex;
        align-items: flex-end;
        gap: 4px;
        margin-bottom: 12px;
    }
    .scale-row.labeled .band-label {
        flex: 0 0 auto;
        width: 44px;
        margin: 0 0 22px;
        text-align: right;
    }
    .scale-main {
        flex: 1;
        min-width: 0;
    }
    .scale-row:last-child {
        margin-bottom: 0;
    }
    .band-label {
        font-size: var(--smFontSize);
        font-weight: 600;
        color: var(--txtHintColor);
        margin-bottom: 6px;
    }
    /* 24px badge + 18px stem sit above an 8px track: badges are top:0, the
       track's top is offset by exactly that height so the stems (drawn as each
       badge's ::after) touch it. */
    .scale-plot {
        position: relative;
        height: 50px;
        margin: 0 12px;
    }
    .scale-track {
        position: absolute;
        top: 42px;
        left: 0;
        right: 0;
        height: 8px;
        border-radius: 4px;
    }
    .badge {
        position: absolute;
        top: 0;
        transform: translateX(-50%);
        width: 24px;
        height: 24px;
        border-radius: 50%;
        border: 2px solid var(--baseColor);
        box-shadow: 0 1px 3px var(--shadowColor);
        color: #fff;
        font-size: var(--xsFontSize);
        font-weight: 600;
        line-height: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        padding: 0;
    }
    .badge::after {
        content: "";
        position: absolute;
        top: 100%;
        left: 50%;
        width: 2px;
        height: 18px;
        background: var(--txtHintColor);
        opacity: 0.5;
        transform: translateX(-50%);
    }
    .scale-axis {
        position: relative;
        height: 16px;
        margin: 6px 12px 0;
    }
    .tick {
        position: absolute;
        transform: translateX(-50%);
        font-size: var(--xsFontSize);
        color: var(--txtHintColor);
    }
</style>
