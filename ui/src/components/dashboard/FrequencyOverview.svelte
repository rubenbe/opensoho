<script>
    import { onMount } from "svelte";
    import { scale } from "svelte/transition";
    import ApiClient from "@/utils/ApiClient";
    import {
        BANDS,
        BAND_LABELS,
        BAND_WIDTHS,
        STANDARD_CHANNELS,
        bondingGroups,
        frequencyToBand,
        frequencyToChannel,
        htmodeWidth,
    } from "@/utils/frequencies";

    let isLoading = false;
    let selectedScope = "healthy"; // "healthy" | "all" | <device id>
    let devices = [];
    let bands = []; // processed render model, see buildBand()

    // Does `flags` forbid the given channel width? Mirrors validateRadioHtModeFlags in opensoho.go.
    function widthForbidden(width, flags) {
        if (!flags) return false;
        switch (width) {
            case 40:
                return flags.has("no_ht40-") && flags.has("no_ht40+");
            case 80:
                return flags.has("no_80mhz");
            case 160:
                return flags.has("no_160mhz");
            case 20:
                return flags.has("no_20mhz");
            default:
                return false;
        }
    }

    // Returns the render model for one band, or null when there is nothing to show.
    function buildBand(band, radios, freqRows, deviceNameById) {
        // Hardware-advertised frequencies (union across scope) + merged flags per frequency.
        const validFreqs = new Set();
        const flagsByFreq = {};
        for (const r of freqRows) {
            if (frequencyToBand(r.frequency) !== band) continue;
            validFreqs.add(r.frequency);
            const flags = flagsByFreq[r.frequency] || (flagsByFreq[r.frequency] = new Set());
            for (const f of r.flags || []) flags.add(f);
        }
        const hasFreqData = validFreqs.size > 0;

        // Configured radios: widths in use per frequency + which devices use them (for tooltips).
        const usedWidths = {}; // freq -> Set(width)
        const usedDevices = {}; // `${freq}:${width}` -> Set(device name)
        let hasRadios = false;
        for (const radio of radios) {
            if (frequencyToBand(radio.frequency) !== band) continue;
            hasRadios = true;
            const width = htmodeWidth(radio.htmode);
            if (width == null) continue;
            (usedWidths[radio.frequency] || (usedWidths[radio.frequency] = new Set())).add(width);
            const key = `${radio.frequency}:${width}`;
            const name = deviceNameById[radio.device] || radio.device || "?";
            (usedDevices[key] || (usedDevices[key] = new Set())).add(name);
        }

        // Skip bands with nothing configured and no advertised frequencies (keeps e.g. an empty
        // 6 GHz plan from cluttering the card).
        if (!hasRadios && !hasFreqData) return null;

        const tiers = (BAND_WIDTHS[band] || []).map((width) => {
            const groups = bondingGroups(band, width).map((g) => {
                const channels = g.frequencies.map((f) => frequencyToChannel(f));
                const mergedFlags = new Set();
                for (const f of g.frequencies) for (const fl of flagsByFreq[f] || []) mergedFlags.add(fl);

                const used = g.frequencies.some((f) => usedWidths[f]?.has(width));
                const missing = hasFreqData && g.frequencies.some((f) => !validFreqs.has(f));
                const invalid = !g.complete || missing || widthForbidden(width, mergedFlags);

                const state = used ? "used" : invalid ? "invalid" : "available";
                const label =
                    g.span === 1
                        ? `${channels[0]}`
                        : `${channels[0]}–${channels[channels.length - 1]}`;

                const devs = new Set();
                for (const f of g.frequencies) {
                    for (const d of usedDevices[`${f}:${width}`] || []) devs.add(d);
                }
                const flagList = Array.from(mergedFlags);
                const title =
                    `${width} MHz · ch ${channels.join("+")} · ${g.frequencies.join("/")} MHz` +
                    (devs.size ? ` · in use: ${Array.from(devs).join(", ")}` : invalid ? " · invalid" : "") +
                    (flagList.length ? ` · flags: ${flagList.join(", ")}` : "");

                return {
                    key: `${width}:${g.startIndex}`,
                    startIndex: g.startIndex,
                    span: g.span,
                    state,
                    label,
                    title,
                };
            });
            return { width, groups };
        });

        return {
            band,
            label: BAND_LABELS[band] || band,
            cols: (STANDARD_CHANNELS[band] || []).length,
            tiers,
        };
    }

    export async function load() {
        isLoading = true;
        try {
            const [deviceRows, radios, freqRows] = await Promise.all([
                ApiClient.collection("devices").getFullList({
                    fields: "id,name,health_status",
                    sort: "name",
                    requestKey: "freq_overview_devices",
                }),
                ApiClient.collection("radios").getFullList({
                    fields: "device,frequency,htmode",
                    requestKey: "freq_overview_radios",
                }),
                ApiClient.collection("radio_frequencies").getFullList({
                    fields: "device,frequency,flags",
                    requestKey: "freq_overview_freqs",
                }),
            ]);

            devices = deviceRows;
            const deviceNameById = {};
            for (const d of deviceRows) deviceNameById[d.id] = d.name;

            // Allowed device-id set for the current scope (null = no filtering).
            let allowed = null;
            if (selectedScope === "healthy") {
                allowed = new Set(deviceRows.filter((d) => d.health_status === "healthy").map((d) => d.id));
            } else if (selectedScope !== "all") {
                allowed = new Set([selectedScope]);
            }
            const inScope = (row) => allowed === null || allowed.has(row.device);
            const scopedRadios = radios.filter(inScope);
            const scopedFreqs = freqRows.filter(inScope);

            bands = BANDS.map((b) => buildBand(b, scopedRadios, scopedFreqs, deviceNameById)).filter(Boolean);
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
                                {#each tier.groups as g (g.key)}
                                    <div
                                        class="block {g.state}"
                                        style="grid-column:{g.startIndex + 1} / span {g.span}"
                                        title={g.title}
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
