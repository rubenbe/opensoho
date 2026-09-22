<script>
    import { onDestroy } from "svelte";
    import { pageTitle } from "@/stores/app";
    import PageWrapper from "@/components/base/PageWrapper.svelte";
    import RefreshButton from "@/components/base/RefreshButton.svelte";
    import Field from "@/components/base/Field.svelte";
    import CardMenu from "@/components/dashboard/CardMenu.svelte";
    import DeviceHealthChart from "@/components/dashboard/DeviceHealthChart.svelte";
    import ClientsPerDeviceChart from "@/components/dashboard/ClientsPerDeviceChart.svelte";
    import ClientsPerChannelChart from "@/components/dashboard/ClientsPerChannelChart.svelte";
    import ClientSignalQualityChart from "@/components/dashboard/ClientSignalQualityChart.svelte";
    import FrequencyOverview from "@/components/dashboard/FrequencyOverview.svelte";
    import NetworkOverview from "@/components/dashboard/NetworkOverview.svelte";
    import WifiVersionChart from "@/components/dashboard/WifiVersionChart.svelte";

    $pageTitle = "Dashboard";

    let deviceHealthChart;
    let clientsPerDeviceChart;
    let clientsPerChannelChart;
    let clientSignalQualityChart;
    let frequencyOverview;
    let networkOverview;
    let wifiVersionChart;

    // Remember the "Client Signal Quality" per-band view across page loads.
    const SIGNAL_SPLIT_STORAGE_KEY = "dashboard_signal_split_by_band";
    let signalSplitByBand = false;
    try {
        signalSplitByBand = window.localStorage?.getItem(SIGNAL_SPLIT_STORAGE_KEY) === "1";
    } catch (_) {}

    function saveSignalSplitByBand(value) {
        try {
            window.localStorage?.setItem(SIGNAL_SPLIT_STORAGE_KEY, value ? "1" : "0");
        } catch (_) {}
    }

    function refreshAll() {
        deviceHealthChart?.load();
        clientsPerDeviceChart?.load();
        clientsPerChannelChart?.load();
        clientSignalQualityChart?.load();
        frequencyOverview?.load();
        networkOverview?.load();
        wifiVersionChart?.load();
    }

    const refreshInterval = setInterval(refreshAll, 15000);

    onDestroy(() => clearInterval(refreshInterval));
</script>

<PageWrapper>
    <header class="page-header">
        <nav class="breadcrumbs">
            <div class="breadcrumb-item">{$pageTitle}</div>
        </nav>

        <div class="inline-flex gap-5">
            <RefreshButton on:refresh={refreshAll} />
        </div>
    </header>

    <div class="dashboard-grid">
        <div class="dashboard-card">
            <h6 class="card-title">Device Health</h6>
            <DeviceHealthChart bind:this={deviceHealthChart} />
        </div>
        <div class="dashboard-card">
            <h6 class="card-title">Clients per Device</h6>
            <ClientsPerDeviceChart bind:this={clientsPerDeviceChart} />
        </div>
        <div class="dashboard-card">
            <h6 class="card-title">Clients per Channel</h6>
            <ClientsPerChannelChart bind:this={clientsPerChannelChart} />
        </div>
        <div class="dashboard-card">
            <h6 class="card-title">AP WiFi Versions</h6>
            <WifiVersionChart bind:this={wifiVersionChart} />
        </div>
        <div class="dashboard-card">
            <div class="card-header">
                <h6 class="card-title">Client Signal Quality</h6>
                <CardMenu label="Client signal quality options">
                    <Field class="form-field form-field-sm form-field-toggle m-0 p-5" let:uniqueId>
                        <input
                            type="checkbox"
                            id={uniqueId}
                            bind:checked={signalSplitByBand}
                            on:change={(e) => saveSignalSplitByBand(e.currentTarget.checked)}
                        />
                        <label for={uniqueId}>Split per frequency band</label>
                    </Field>
                </CardMenu>
            </div>
            <ClientSignalQualityChart bind:this={clientSignalQualityChart} splitByBand={signalSplitByBand} />
        </div>
        <div class="dashboard-card wide">
            <h6 class="card-title">Frequency Overview</h6>
            <FrequencyOverview bind:this={frequencyOverview} />
        </div>
        <div class="dashboard-card wide">
            <h6 class="card-title">Network Overview</h6>
            <NetworkOverview bind:this={networkOverview} />
        </div>
    </div>
</PageWrapper>

<style>
    .dashboard-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
        gap: var(--baseSpacing);
        padding-top: var(--baseSpacing);
    }
    .dashboard-card {
        background: var(--baseColor);
        border-radius: var(--lgRadius);
        padding: var(--baseSpacing);
        box-shadow: 0 1px 4px var(--shadowColor);
    }
    .dashboard-card.wide {
        grid-column: 1 / -1;
    }
    .card-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: var(--xsSpacing);
        margin: 0 0 var(--smSpacing);
    }
    .card-header .card-title {
        margin: 0;
    }
    .card-title {
        margin: 0 0 var(--smSpacing);
        font-size: var(--lgFontSize);
        font-weight: 600;
        color: var(--txtPrimaryColor);
    }
</style>
