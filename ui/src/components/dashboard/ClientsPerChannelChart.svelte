<script>
    import { onMount } from "svelte";
    import ApiClient from "@/utils/ApiClient";
    import { push } from "svelte-spa-router";
    import BandPieCharts from "@/components/dashboard/BandPieCharts.svelte";

    const PALETTE = [
        "#4e79a7", "#f28e2b", "#e15759", "#76b7b2", "#59a14f",
        "#edc948", "#b07aa1", "#ff9da7", "#9c755f", "#bab0ac",
    ];

    const BANDS = [
        { key: "2.4", label: "2.4 GHz" },
        { key: "5", label: "5 GHz" },
        { key: "6", label: "6 GHz" },
    ];

    let isLoading = false;
    let series = [];

    function buildSeries(band, label, channelCounts) {
        const channels = Object.keys(channelCounts).map(Number).sort((a, b) => a - b);
        const slices = channels.map((c, i) => ({
            label: `Ch ${c}`,
            value: channelCounts[c],
            color: PALETTE[i % PALETTE.length],
            channel: c,
        }));
        return { key: band, label, slices };
    }

    export async function load() {
        isLoading = true;
        try {
            const records = await ApiClient.collection("connected_clients").getFullList({
                fields: "channel,band",
                requestKey: "clients_per_channel",
            });

            const counts = { "2.4": {}, "5": {}, "6": {} };
            for (const r of records) {
                if (!r.channel || !counts[r.band]) continue;
                counts[r.band][r.channel] = (counts[r.band][r.channel] || 0) + 1;
            }

            series = BANDS
                // 2.4 and 5 GHz always render; 6 GHz only when it has clients.
                .filter((b) => b.key !== "6" || Object.keys(counts["6"]).length > 0)
                .map((b) => buildSeries(b.key, b.label, counts[b.key]));
        } catch (err) {
            if (!err?.isAbort) {
                ApiClient.error(err);
            }
        } finally {
            isLoading = false;
        }
    }

    function onSelect(e) {
        const { band, slice } = e.detail;
        const filter = `channel = ${slice.channel} && band = "${band}"`;
        push(`/collections?collection=connected_clients&filter=${encodeURIComponent(filter)}`);
    }

    onMount(() => {
        load();
    });
</script>

<BandPieCharts {series} {isLoading} on:select={onSelect} />
