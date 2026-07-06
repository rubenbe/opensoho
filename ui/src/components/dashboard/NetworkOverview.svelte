<script>
    import { onMount } from "svelte";
    import { scale } from "svelte/transition";
    import ApiClient from "@/utils/ApiClient";

    let isLoading = false;
    let selectedScope = ""; // the device id currently shown
    let devices = [];
    let ports = [];

    // Only show the PoE column when this device actually reports PoE on some port.
    $: hasPoe = ports.some((p) => p.poe != null);

    // Any LLDP neighbour on any port means lldpd is reporting; none likely means
    // lldpd isn't installed/running on the device.
    $: hasLldp = ports.some((p) => p.neighbors && p.neighbors.length);

    // LuCI package-manager page of the selected device, so the warning can link
    // straight to where lldpd can be installed. Empty when the device has no IP.
    $: selectedIp = devices.find((d) => d.id === selectedScope)?.ip || "";
    $: luciPackagesUrl = selectedIp
        ? `http://${selectedIp}/cgi-bin/luci/admin/system/package-manager`
        : "";

    export async function load() {
        isLoading = true;
        try {
            const res = await ApiClient.send("/api/v1/network-overview", {
                method: "GET",
                query: { device: selectedScope },
                requestKey: "network_overview",
            });
            devices = res.devices || [];
            ports = res.ports || [];
            // Adopt the server-resolved scope so the selector syncs on first load.
            if (res.scope) {
                selectedScope = res.scope;
            }
        } catch (err) {
            if (!err?.isAbort) {
                ApiClient.error(err);
            }
        } finally {
            isLoading = false;
        }
    }

    // Jump the card to the neighbour device (stays on the dashboard).
    function openDevice(id) {
        selectedScope = id;
        load();
    }

    onMount(() => {
        load();
    });
</script>

<div class="net-overview" class:loading={isLoading}>
    {#if isLoading}
        <div class="net-loader loader" transition:scale={{ duration: 150 }} />
    {/if}

    <div class="net-toolbar">
        <label class="net-device-select">
            <span>Device</span>
            <select bind:value={selectedScope} on:change={load}>
                {#each devices as d (d.id)}
                    <option value={d.id}>{d.name || d.id}</option>
                {/each}
            </select>
        </label>

        {#if !isLoading && !hasLldp}
            {#if luciPackagesUrl}
                <a
                    class="net-warning"
                    href={luciPackagesUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    title="Open the device's LuCI package manager"
                >
                    ⚠️ No LLDP data, verify whether lldpd is installed on this device.
                </a>
            {:else}
                <span class="net-warning">
                    ⚠️ No LLDP data, verify whether lldpd is installed on this device.
                </span>
            {/if}
        {/if}
    </div>

    {#if ports.length === 0}
        <div class="net-empty">No ports known for this device.</div>
    {:else}
        <div class="net-scroll">
            <table class="net-table">
                <thead>
                    <tr>
                        <th>Port</th>
                        <th>Link</th>
                        {#if hasPoe}
                            <th>PoE</th>
                        {/if}
                        <th>Bridge</th>
                        <th>Neighbours</th>
                    </tr>
                </thead>
                <tbody>
                    {#each ports as p (p.port)}
                        <tr>
                            <td class="net-port">{p.port}</td>
                            {#if p.speed}
                                <td>
                                    <span class="net-dot net-dot--up" />{p.speed}
                                </td>
                            {:else}
                                <td class="net-muted">
                                    <span class="net-dot" />no link
                                </td>
                            {/if}
                            {#if hasPoe}
                                <td class="net-muted">
                                    {#if p.poe}
                                        ⚡ {p.poe.toFixed(1)}W
                                    {:else if p.poe != null}
                                        <span
                                            class="net-poe-idle"
                                            title="PoE port, not supplying power"
                                        >⚡</span> —
                                    {:else}
                                        —
                                    {/if}
                                </td>
                            {/if}
                            <td class="net-muted">{p.bridge || "—"}</td>
                            <td>
                                {#if p.neighbors && p.neighbors.length}
                                    {#each p.neighbors as n (n.mac + "-" + n.name)}
                                        <div class="net-neighbour">
                                            {#if n.knownDeviceId}
                                                <button
                                                    type="button"
                                                    class="device-link"
                                                    on:click={() => openDevice(n.knownDeviceId)}
                                                >
                                                    {n.name || "(unnamed)"} ({n.mac})
                                                </button>
                                            {:else}
                                                <span>{n.name || "(unnamed)"} ({n.mac})</span>
                                            {/if}
                                        </div>
                                    {/each}
                                {:else}
                                    <span class="net-muted">—</span>
                                {/if}
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    {/if}
</div>

<style>
    .net-overview {
        position: relative;
        width: 100%;
        min-height: 120px;
    }
    .net-overview.loading {
        opacity: 0.6;
        pointer-events: none;
    }
    .net-loader {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        z-index: 2;
    }

    .net-toolbar {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: var(--xsSpacing);
        margin-bottom: var(--smSpacing);
    }
    .net-device-select {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        font-size: var(--smFontSize);
        color: var(--txtHintColor);
    }
    .net-device-select select {
        font-size: var(--smFontSize);
        padding: 4px 8px;
        border-radius: var(--baseRadius);
        border: 1px solid var(--baseAlt2Color);
        background: var(--baseColor);
        color: var(--txtPrimaryColor);
    }

    .net-scroll {
        overflow-x: auto;
    }
    .net-table {
        width: 100%;
        border-collapse: collapse;
        font-size: var(--smFontSize);
    }
    .net-table th {
        text-align: left;
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.03em;
        color: var(--txtHintColor);
        padding: 4px 8px;
        border-bottom: 1px solid var(--baseAlt2Color);
    }
    .net-table td {
        padding: 6px 8px;
        border-bottom: 1px solid var(--baseAlt1Color);
        color: var(--txtPrimaryColor);
        vertical-align: top;
    }
    .net-port {
        font-family: var(--monospaceFontFamily, monospace);
        color: var(--txtPrimaryColor);
        white-space: nowrap;
    }
    .net-muted {
        color: var(--txtHintColor);
    }
    .net-dot {
        display: inline-block;
        width: 8px;
        height: 8px;
        border-radius: 50%;
        margin-right: 6px;
        vertical-align: middle;
        background: var(--txtHintColor);
    }
    .net-dot--up {
        background: var(--successColor);
    }
    .net-poe-idle {
        filter: grayscale(1);
        opacity: 0.4;
        cursor: help;
    }
    .net-warning {
        font-size: var(--smFontSize);
        color: var(--txtHintColor);
        text-decoration: none;
    }
    a.net-warning {
        cursor: pointer;
    }
    a.net-warning:hover {
        text-decoration: underline;
    }
    .net-neighbour + .net-neighbour {
        margin-top: 2px;
    }

    .net-empty {
        font-size: var(--smFontSize);
        color: var(--txtHintColor);
        padding: var(--smSpacing) 0;
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
