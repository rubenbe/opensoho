<script>
    import { onMount } from "svelte";
    import { scale } from "svelte/transition";
    import ApiClient from "@/utils/ApiClient";

    let isLoading = false;
    let selectedScope = "healthy"; // "healthy" | "all" | <device id>
    let devices = [];
    let bands = []; // server-computed render model (see /api/v1/frequency-overview)

    // Compose the hover tooltip from a block's structured fields.
    function blockTitle(width, g) {
        const chans = (g.channels || []).join("+");
        const freqs = (g.frequencies || []).join("/");
        let t = `${width} MHz · ch ${chans} · ${freqs} MHz`;
        if (g.devices?.length) {
            t += ` · in use: ${g.devices.join(", ")}`;
        } else if (g.state === "invalid") {
            t += " · invalid";
        }
        if (g.flags?.length) {
            t += ` · flags: ${g.flags.join(", ")}`;
        }
        return t;
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
                                        title={blockTitle(tier.width, g)}
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
</style>
