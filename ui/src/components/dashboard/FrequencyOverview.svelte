<script>
    import { onMount } from "svelte";
    import { scale } from "svelte/transition";
    import ApiClient from "@/utils/ApiClient";
    import {
        BANDS,
        BAND_LABELS,
        STANDARD_CHANNELS,
        frequencyToBand,
        htmodeWidth,
    } from "@/utils/frequencies";

    let isLoading = false;
    let selectedDevice = ""; // "" = all devices
    let devices = [];
    let bands = []; // processed render model, see buildBands()

    // Returns the render model for one band, or null when there is nothing to show.
    function buildBand(band, radios, freqRows, deviceNameById) {
        const plan = STANDARD_CHANNELS[band] || [];
        const stdFreqs = new Set(plan.map((c) => c.frequency));

        // Frequencies the hardware advertised for this band (union across scope),
        // plus the merged flags per frequency.
        const validFreqs = new Set();
        const flagsByFreq = {};
        for (const r of freqRows) {
            if (frequencyToBand(r.frequency) !== band) continue;
            validFreqs.add(r.frequency);
            const flags = flagsByFreq[r.frequency] || (flagsByFreq[r.frequency] = new Set());
            for (const f of r.flags || []) flags.add(f);
        }
        const hasFreqData = validFreqs.size > 0;

        // Configured radios for this band: which widths are in use per frequency,
        // and which devices configured them (for tooltips).
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

        // Skip bands with nothing configured and no advertised frequencies to keep
        // the card from being cluttered with empty plans (e.g. 6 GHz).
        if (!hasRadios && !hasFreqData) return null;

        const flagsList = (freq) => Array.from(flagsByFreq[freq] || []);
        const devicesFor = (freq, width) => Array.from(usedDevices[`${freq}:${width}`] || []);

        const channels = plan.map(({ channel, frequency }) => {
            const flags = flagsByFreq[frequency];
            // A channel is invalid when the hardware reported its frequencies but
            // this one is not among them, or it is explicitly flagged no_20mhz.
            const channelInvalid =
                (hasFreqData && !validFreqs.has(frequency)) ||
                (flags && flags.has("no_20mhz"));

            // 40 MHz needs an adjacent 20 MHz channel in the plan and must not be
            // forbidden by both ht40 flags (matches opensoho.go validation).
            const canPair = stdFreqs.has(frequency + 20) || stdFreqs.has(frequency - 20);
            const ht40Forbidden = flags && flags.has("no_ht40-") && flags.has("no_ht40+");

            const ht20Used = usedWidths[frequency]?.has(20);
            const ht40Used = usedWidths[frequency]?.has(40);

            const ht20State = ht20Used ? "used" : channelInvalid ? "invalid" : "available";
            const ht40State = ht40Used
                ? "used"
                : channelInvalid || !canPair || ht40Forbidden
                  ? "invalid"
                  : "available";

            const freqTxt = `ch ${channel} · ${frequency} MHz`;
            const flagTxt = flagsList(frequency).length ? ` · flags: ${flagsList(frequency).join(", ")}` : "";
            const ht20Dev = devicesFor(frequency, 20);
            const ht40Dev = devicesFor(frequency, 40);

            return {
                channel,
                frequency,
                ht20State,
                ht40State,
                ht20Title:
                    `HT20 · ${freqTxt}` +
                    (ht20Dev.length ? ` · in use: ${ht20Dev.join(", ")}` : ht20State === "invalid" ? " · invalid" : "") +
                    flagTxt,
                ht40Title:
                    `HT40 · ${freqTxt}` +
                    (ht40Dev.length ? ` · in use: ${ht40Dev.join(", ")}` : ht40State === "invalid" ? " · invalid" : "") +
                    flagTxt,
            };
        });

        return { band, label: BAND_LABELS[band] || band, channels };
    }

    export async function load() {
        isLoading = true;
        try {
            const filter = selectedDevice ? `device = "${selectedDevice}"` : "";

            const [deviceRows, radios, freqRows] = await Promise.all([
                ApiClient.collection("devices").getFullList({
                    fields: "id,name",
                    sort: "name",
                    requestKey: "freq_overview_devices",
                }),
                ApiClient.collection("radios").getFullList({
                    fields: "device,radio,frequency,htmode,enabled",
                    filter,
                    requestKey: "freq_overview_radios",
                }),
                ApiClient.collection("radio_frequencies").getFullList({
                    fields: "device,radio,channel,frequency,flags",
                    filter,
                    requestKey: "freq_overview_freqs",
                }),
            ]);

            devices = deviceRows;
            const deviceNameById = {};
            for (const d of deviceRows) deviceNameById[d.id] = d.name;

            bands = BANDS.map((b) => buildBand(b, radios, freqRows, deviceNameById)).filter(Boolean);
        } catch (err) {
            if (!err?.isAbort) {
                ApiClient.error(err);
            }
        } finally {
            isLoading = false;
        }
    }

    function onDeviceChange() {
        load();
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
            <span>Device</span>
            <select bind:value={selectedDevice} on:change={onDeviceChange}>
                <option value="">All devices</option>
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
                <div class="band-rows">
                    <div class="square-row">
                        <span class="row-label">HT20</span>
                        <div class="squares">
                            {#each b.channels as c (c.frequency)}
                                <div class="square {c.ht20State}" title={c.ht20Title}>
                                    <span class="ch-num">{c.channel}</span>
                                </div>
                            {/each}
                        </div>
                    </div>
                    <div class="square-row">
                        <span class="row-label">HT40</span>
                        <div class="squares ht40">
                            {#each b.channels as c (c.frequency)}
                                <div class="square {c.ht40State}" title={c.ht40Title} />
                            {/each}
                        </div>
                    </div>
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
        z-index: 1;
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
        border: 1px solid var(--baseAlt2Color, #e0e0e0);
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
    .band-rows {
        display: flex;
        flex-direction: column;
        gap: var(--cell-gap);
        width: max-content;
    }
    .square-row {
        display: flex;
        align-items: center;
        gap: var(--cell-gap);
    }
    .row-label {
        flex: 0 0 auto;
        width: 40px;
        font-size: 10px;
        font-weight: 600;
        color: var(--txtHintColor);
    }
    .squares {
        display: flex;
        gap: var(--cell-gap);
    }
    /* Offset the HT40 row by half a cell so each 40 MHz block visually straddles
       the two 20 MHz channels it overlaps. */
    .squares.ht40 {
        margin-left: calc((var(--cell) + var(--cell-gap)) / 2);
    }
    .square {
        width: var(--cell);
        height: var(--cell);
        border-radius: 2px;
        display: flex;
        align-items: center;
        justify-content: center;
        box-sizing: border-box;
        flex: 0 0 auto;
    }
    .ch-num {
        font-size: 10px;
        line-height: 1;
        color: var(--txtHintColor);
    }
    .square.used .ch-num {
        color: #fff;
    }

    .square.used,
    .swatch.used {
        background: var(--successColor);
    }
    .square.available,
    .swatch.available {
        background: var(--baseColor);
        border: 1px solid var(--successColor);
    }
    .square.invalid,
    .swatch.invalid {
        background: var(--baseAlt1Color, #f0f0f0);
        border: 1px solid var(--baseAlt2Color, #e0e0e0);
    }
    .square.invalid .ch-num {
        color: var(--txtDisabledColor);
    }

    .freq-empty {
        font-size: var(--smFontSize);
        color: var(--txtHintColor);
        padding: var(--smSpacing) 0;
    }
</style>
