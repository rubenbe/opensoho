<script>
    import { onDestroy } from "svelte";
    import { pageTitle } from "@/stores/app";
    import PageWrapper from "@/components/base/PageWrapper.svelte";
    import RefreshButton from "@/components/base/RefreshButton.svelte";
    import DeviceHealthChart from "@/components/dashboard/DeviceHealthChart.svelte";
    import ClientsPerDeviceChart from "@/components/dashboard/ClientsPerDeviceChart.svelte";
    import ClientsPerChannelChart from "@/components/dashboard/ClientsPerChannelChart.svelte";
    import ClientSignalQualityChart from "@/components/dashboard/ClientSignalQualityChart.svelte";
    import FrequencyOverview from "@/components/dashboard/FrequencyOverview.svelte";

    $pageTitle = "Dashboard";

    let deviceHealthChart;
    let clientsPerDeviceChart;
    let clientsPerChannelChart;
    let clientSignalQualityChart;
    let frequencyOverview;

    function refreshAll() {
        deviceHealthChart?.load();
        clientsPerDeviceChart?.load();
        clientsPerChannelChart?.load();
        clientSignalQualityChart?.load();
        frequencyOverview?.load();
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
            <h6 class="card-title">Client Signal Quality</h6>
            <ClientSignalQualityChart bind:this={clientSignalQualityChart} />
        </div>
        <div class="dashboard-card wide">
            <h6 class="card-title">Frequency Overview</h6>
            <FrequencyOverview bind:this={frequencyOverview} />
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
    .card-title {
        margin: 0 0 var(--smSpacing);
        font-size: var(--lgFontSize);
        font-weight: 600;
        color: var(--txtPrimaryColor);
    }
</style>
