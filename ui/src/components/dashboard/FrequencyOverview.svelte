<script>
    import { onMount, onDestroy, tick } from "svelte";
    import { scale } from "svelte/transition";
    import { push } from "svelte-spa-router";
    import ApiClient from "@/utils/ApiClient";

    let isLoading = false;
    let selectedScope = "healthy"; // "healthy" | "all" | <device id>
    let devices = [];
    let bands = []; // server-computed render model (see /api/v1/frequency-overview)

    const STATE_LABELS = { used: "In use", available: "Available", invalid: "Invalid" };

    // Custom tooltip state.
    let tip = null; // { x, y, width, block }
    let tipEl;
    let hideTimer;

    async function showTip(node, band, width, block) {
        clearTimeout(hideTimer);
        const rect = node.getBoundingClientRect();
        tip = { x: rect.left, y: rect.bottom + 6, freqMin: band.freqMin, freqMax: band.freqMax, width, block };
        // Clamp to the viewport once the tooltip has rendered and we know its size.
        await tick();
        if (!tipEl || !tip) return;
        const t = tipEl.getBoundingClientRect();
        const margin = 8;
        let x = tip.x;
        let y = tip.y;
        if (x + t.width > window.innerWidth - margin) x = window.innerWidth - t.width - margin;
        if (x < margin) x = margin;
        if (y + t.height > window.innerHeight - margin) y = rect.top - t.height - 6;
        tip = { ...tip, x, y };
    }

    function scheduleHide() {
        clearTimeout(hideTimer);
        hideTimer = setTimeout(() => (tip = null), 120);
    }

    function openRadios(id, freqMin, freqMax) {
        tip = null;
        const filter = `device="${id}" && frequency >= ${freqMin} && frequency <= ${freqMax}`;
        push("/collections?collection=radios&filter=" + encodeURIComponent(filter));
    }

    export async function load() {
        isLoading = true;
        try {
            const res = await ApiClient.send("/api/v1/frequency-overview", {
                method: "GET",
                query: { scope: selectedScope },
                requestKey: "frequency_overview",
            });
            devices = res.devices || [];
            bands = res.bands || [];
        } catch (err) {
            if (!err?.isAbort) {
                ApiClient.error(err);
            }
        } finally {
            isLoading = false;
        }
    }

    onMount(() => {
        load();
    });

    onDestroy(() => clearTimeout(hideTimer));
</script>

<div class="freq-overview" class:loading={isLoading}>
    {#if isLoading}
        <div class="freq-loader loader" transition:scale={{ duration: 150 }} />
    {/if}

    <div class="freq-toolbar">
        <label class="freq-device-select">
            <span>Devices</span>
            <select bind:value={selectedScope} on:change={load}>
                <option value="healthy">All healthy devices</option>
                <option value="all">All devices</option>
                {#each devices as d (d.id)}
                    <option value={d.id}>{d.name || d.id}</option>
                {/each}
            </select>
        </label>

        <div class="freq-legend">
            <span class="legend-item"><span class="swatch used" /> In use</span>
            <span class="legend-item"><span class="swatch available" /> Available</span>
            <span class="legend-item"><span class="swatch invalid" /> Invalid</span>
        </div>
    </div>

    {#if bands.length === 0}
        <div class="freq-empty">No radios configured.</div>
    {/if}

    {#each bands as b (b.band)}
        <div class="band-block">
            <div class="band-label">{b.label}</div>
            <div class="band-scroll">
                <div class="band-grid">
                    {#each b.tiers as tier (tier.width)}
                        <div class="tier">
                            <span class="tier-label">{tier.width}</span>
                            <div class="tier-cells" style="--cols:{b.cols}">
                                {#each tier.groups as g (g.startIndex)}
                                    <div
                                        class="block {g.state}"
                                        style="grid-column:{g.startIndex + 1} / span {g.span}"
                                        on:mouseenter={(e) => showTip(e.currentTarget, b, tier.width, g)}
                                        on:mouseleave={scheduleHide}
                                    >
                                        <span class="block-label">{g.label}</span>
                                    </div>
                                {/each}
                            </div>
                        </div>
                    {/each}
                </div>
            </div>
        </div>
    {/each}

    {#if tip}
        <div
            class="freq-tip"
            bind:this={tipEl}
            style="left:{tip.x}px; top:{tip.y}px"
            on:mouseenter={() => clearTimeout(hideTimer)}
            on:mouseleave={scheduleHide}
        >
            <div class="tip-heading">
                {tip.width} MHz · channel{tip.block.channels.length > 1 ? "s" : ""}
                {tip.block.channels.join(", ")}
            </div>
            <div class="tip-row">
                <span class="tip-label">Status</span>
                <span class="tip-state {tip.block.state}">{STATE_LABELS[tip.block.state] || tip.block.state}</span>
            </div>
            <div class="tip-row">
                <span class="tip-label">Frequencies</span>
                <span>{tip.block.frequencies.join(", ")} MHz</span>
            </div>
            {#if tip.block.devices?.length}
                <div class="tip-row">
                    <span class="tip-label">In use by</span>
                    <span class="tip-devices">
                        {#each tip.block.devices as d (d.id)}
                            <button type="button" class="device-link" on:click={() => openRadios(d.id, tip.freqMin, tip.freqMax)}>{d.name}</button>
                        {/each}
                    </span>
                </div>
            {/if}
            {#if tip.block.supportedBy?.length}
                <div class="tip-row">
                    <span class="tip-label">Supported by</span>
                    <span class="tip-devices">
                        {#each tip.block.supportedBy as d (d.id)}
                            <button type="button" class="device-link" on:click={() => openRadios(d.id, tip.freqMin, tip.freqMax)}>{d.name}</button>
                        {/each}
                    </span>
                </div>
            {/if}
            {#if tip.block.unsupportedBy?.length}
                <div class="tip-row">
                    <span class="tip-label">Unsupported</span>
                    <span>{tip.block.unsupportedBy.map((d) => d.name).join(", ")}</span>
                </div>
            {/if}
        </div>
    {/if}
</div>

<style>
    .freq-overview {
        --cell: 26px;
        --cell-gap: 3px;
        position: relative;
        width: 100%;
        min-height: 120px;
    }
    .freq-overview.loading {
        opacity: 0.6;
        pointer-events: none;
    }
    .freq-loader {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        z-index: 2;
    }

    .freq-toolbar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex-wrap: wrap;
        gap: var(--xsSpacing);
        margin-bottom: var(--smSpacing);
    }
    .freq-device-select {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        font-size: var(--smFontSize);
        color: var(--txtHintColor);
    }
    .freq-device-select select {
        font-size: var(--smFontSize);
        padding: 4px 8px;
        border-radius: var(--baseRadius);
        border: 1px solid var(--baseAlt2Color);
        background: var(--baseColor);
        color: var(--txtPrimaryColor);
    }

    .freq-legend {
        display: inline-flex;
        gap: var(--smSpacing);
        font-size: var(--smFontSize);
        color: var(--txtHintColor);
    }
    .legend-item {
        display: inline-flex;
        align-items: center;
        gap: 6px;
    }
    .swatch {
        display: inline-block;
        width: 12px;
        height: 12px;
        border-radius: 2px;
    }

    .band-block {
        margin-bottom: var(--smSpacing);
    }
    .band-label {
        font-size: var(--smFontSize);
        font-weight: 600;
        color: var(--txtHintColor);
        margin-bottom: 6px;
    }
    .band-scroll {
        overflow-x: auto;
        padding-bottom: 4px;
    }
    .band-grid {
        display: flex;
        flex-direction: column;
        gap: var(--cell-gap);
        width: max-content;
    }
    .tier {
        display: flex;
        align-items: center;
        gap: var(--cell-gap);
    }
    .tier-label {
        position: sticky;
        left: 0;
        z-index: 1;
        flex: 0 0 auto;
        width: 34px;
        text-align: right;
        padding-right: 6px;
        font-size: 10px;
        font-weight: 600;
        color: var(--txtHintColor);
        background: var(--baseColor);
    }
    .tier-cells {
        display: grid;
        grid-template-columns: repeat(var(--cols), var(--cell));
        gap: var(--cell-gap);
    }
    .block {
        height: var(--cell);
        border-radius: 2px;
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
        box-sizing: border-box;
    }
    .block-label {
        font-size: 10px;
        line-height: 1;
        white-space: nowrap;
        color: var(--txtHintColor);
    }
    .block.used .block-label {
        color: #fff;
    }
    .block.invalid .block-label {
        color: var(--txtDisabledColor);
    }

    .block.used,
    .swatch.used {
        background: var(--successColor);
    }
    .block.available,
    .swatch.available {
        background: var(--baseColor);
        border: 1px solid var(--successColor);
    }
    .block.invalid,
    .swatch.invalid {
        background: var(--baseAlt1Color);
        border: 1px solid var(--baseAlt2Color);
    }

    .freq-empty {
        font-size: var(--smFontSize);
        color: var(--txtHintColor);
        padding: var(--smSpacing) 0;
    }

    .freq-tip {
        position: fixed;
        z-index: 1000;
        min-width: 200px;
        max-width: 320px;
        padding: 10px 12px;
        background: var(--baseColor);
        border: 1px solid var(--baseAlt2Color);
        border-radius: var(--baseRadius);
        box-shadow: 0 2px 10px var(--shadowColor);
        font-size: 13px;
        color: var(--txtPrimaryColor);
    }
    .tip-heading {
        font-weight: 600;
        margin-bottom: 6px;
    }
    .tip-row {
        display: flex;
        gap: 8px;
        margin-top: 4px;
        line-height: 1.4;
    }
    .tip-label {
        flex: 0 0 84px;
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.03em;
        color: var(--txtHintColor);
        padding-top: 1px;
    }
    .tip-state.used {
        color: var(--successColor);
    }
    .tip-state.invalid {
        color: var(--txtDisabledColor);
    }
    .tip-devices {
        display: flex;
        flex-wrap: wrap;
        gap: 4px 10px;
    }
    .device-link {
        padding: 0;
        border: 0;
        background: none;
        font: inherit;
        color: var(--infoColor);
        cursor: pointer;
        text-decoration: none;
    }
    .device-link:hover {
        text-decoration: underline;
    }
</style>
