<script>
    import { onMount } from "svelte";
    import ApiClient from "@/utils/ApiClient";
    import { push } from "svelte-spa-router";
    import BandPieCharts from "@/components/dashboard/BandPieCharts.svelte";

    // Purely presentational: the band/generation categorization itself comes
    // from the server (apiWifiVersions in opensoho.go), which resolves it the
    // same way a radio is actually configured (resolveRadioBand,
    // htModeGenerationPrefix) instead of re-deriving it here.
    const COLORS = {
        "EHT": "#4e79a7",
        "HE": "#59a14f",
        "VHT": "#edc948",
        "HT": "#f28e2b",
        "": "#a5b0c0",
    };

    let isLoading = false;
    let series = [];

    export async function load() {
        isLoading = true;
        try {
            const res = await ApiClient.send("/api/v1/wifi-versions", {
                method: "GET",
                requestKey: "wifi_versions",
            });

            series = (res.bands || []).map((b) => ({
                key: b.key,
                label: b.label,
                slices: b.slices.map((s) => ({
                    label: s.label,
                    value: s.count,
                    color: COLORS[s.prefix] ?? COLORS[""],
                    filter: s.filter,
                })),
            }));
        } catch (err) {
            if (!err?.isAbort) {
                ApiClient.error(err);
            }
        } finally {
            isLoading = false;
        }
    }

    function onSelect(e) {
        const { slice } = e.detail;
        push(`/collections?collection=radios&filter=${encodeURIComponent(slice.filter)}`);
    }

    onMount(() => {
        load();
    });
</script>

<BandPieCharts {series} {isLoading} on:select={onSelect} />
