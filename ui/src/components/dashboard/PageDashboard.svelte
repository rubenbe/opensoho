<script>
    import { pageTitle } from "@/stores/app";
    import PageWrapper from "@/components/base/PageWrapper.svelte";
    import DeviceHealthChart from "@/components/dashboard/DeviceHealthChart.svelte";
    import ClientsPerDeviceChart from "@/components/dashboard/ClientsPerDeviceChart.svelte";
    import ClientsPerChannelChart from "@/components/dashboard/ClientsPerChannelChart.svelte";

    $pageTitle = "Dashboard";
</script>

<PageWrapper>
    <header class="page-header">
        <nav class="breadcrumbs">
            <div class="breadcrumb-item">{$pageTitle}</div>
        </nav>
    </header>

    <div class="dashboard-grid">
        <div class="dashboard-card">
            <h6 class="card-title">Device Health</h6>
            <DeviceHealthChart />
        </div>
        <div class="dashboard-card">
            <h6 class="card-title">Clients per Device</h6>
            <ClientsPerDeviceChart />
        </div>
        <div class="dashboard-card">
            <h6 class="card-title">Clients per Channel</h6>
            <ClientsPerChannelChart />
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
    .card-title {
        margin: 0 0 var(--smSpacing);
        font-size: var(--lgFontSize);
        font-weight: 600;
        color: var(--txtPrimaryColor);
    }
</style>
