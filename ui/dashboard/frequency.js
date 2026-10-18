import { loader, loading, openRecords } from "./charts.js";

const STATE_LABELS = { used: "In use", available: "Available", invalid: "Invalid" };

/**
 * Frequency overview card, rendered from the server-computed model
 * (see /api/v1/frequency-overview).
 *
 * @return {{el: Element, load: Function}}
 */
export function frequencyOverview() {
    const state = store({ isLoading: false });

    let scope = "healthy"; // "healthy" | "all" | <device id>
    let hideTimer;

    const select = t.select({
        onchange: (e) => {
            scope = e.target.value;
            load();
        },
    });

    const bandsEl = t.div({ className: "freq-bands" });

    const tip = t.div({
        className: "freq-tip",
        hidden: true,
        onmouseenter: () => clearTimeout(hideTimer),
        onmouseleave: scheduleHide,
    });

    function hideTip() {
        clearTimeout(hideTimer);
        tip.hidden = true;
    }

    function scheduleHide() {
        clearTimeout(hideTimer);
        hideTimer = setTimeout(hideTip, 120);
    }

    function openRadios(id, band) {
        hideTip();
        openRecords("radios", `device="${id}" && frequency >= ${band.freqMin} && frequency <= ${band.freqMax}`);
    }

    function deviceLinks(devices, band) {
        return t.span(
            { className: "tip-devices" },
            devices.map((d) => {
                return t.button(
                    { type: "button", className: "device-link", onclick: () => openRadios(d.id, band) },
                    d.name,
                );
            }),
        );
    }

    function showTip(node, band, width, block) {
        clearTimeout(hideTimer);

        // wrapped in a Shablon element, as replaceChildren() would render the nulls
        tip.replaceChildren(t.div(
            null,
            t.div(
                { className: "tip-heading" },
                `${width} MHz · channel${block.channels.length > 1 ? "s" : ""} ${block.channels.join(", ")}`,
            ),
            t.div(
                { className: "tip-row" },
                t.span({ className: "tip-label" }, "Status"),
                t.span({ className: `tip-state ${block.state}` }, STATE_LABELS[block.state] || block.state),
            ),
            t.div(
                { className: "tip-row" },
                t.span({ className: "tip-label" }, "Frequencies"),
                t.span(null, `${block.frequencies.join(", ")} MHz`),
            ),
            block.devices?.length
                ? t.div(
                    { className: "tip-row" },
                    t.span({ className: "tip-label" }, "In use by"),
                    deviceLinks(block.devices, band),
                )
                : null,
            block.supportedBy?.length
                ? t.div(
                    { className: "tip-row" },
                    t.span({ className: "tip-label" }, "Supported by"),
                    deviceLinks(block.supportedBy, band),
                )
                : null,
            block.unsupportedBy?.length
                ? t.div(
                    { className: "tip-row" },
                    t.span({ className: "tip-label" }, "Unsupported"),
                    t.span(null, block.unsupportedBy.map((d) => d.name).join(", ")),
                )
                : null,
        ));

        // position below the block, clamped to the viewport
        const rect = node.getBoundingClientRect();
        tip.style.left = rect.left + "px";
        tip.style.top = rect.bottom + 6 + "px";
        tip.hidden = false;

        const tipRect = tip.getBoundingClientRect();
        const margin = 8;
        let x = rect.left;
        let y = rect.bottom + 6;
        if (x + tipRect.width > window.innerWidth - margin) x = window.innerWidth - tipRect.width - margin;
        if (x < margin) x = margin;
        if (y + tipRect.height > window.innerHeight - margin) y = rect.top - tipRect.height - 6;
        tip.style.left = x + "px";
        tip.style.top = y + "px";
    }

    function render(devices, bands) {
        select.replaceChildren(
            t.option({ value: "healthy" }, "All healthy devices"),
            t.option({ value: "all" }, "All devices"),
            ...devices.map((d) => t.option({ value: d.id }, d.name || d.id)),
        );
        select.value = scope;

        if (!bands.length) {
            bandsEl.replaceChildren(t.div({ className: "freq-empty" }, "No radios configured."));
            return;
        }

        bandsEl.replaceChildren(...bands.map((b) => {
            return t.div(
                { className: "band-block" },
                t.div({ className: "band-label" }, b.label),
                t.div(
                    { className: "band-scroll" },
                    t.div(
                        { className: "band-grid" },
                        b.tiers.map((tier) => {
                            return t.div(
                                { className: "tier" },
                                t.span({ className: "tier-label" }, tier.width),
                                t.div(
                                    { className: "tier-cells", style: `--cols:${b.cols}` },
                                    tier.groups.map((g) => {
                                        return t.div(
                                            {
                                                className: `block ${g.state}`,
                                                style: `grid-column:${g.startIndex + 1} / span ${g.span}`,
                                                onmouseenter: (e) => showTip(e.currentTarget, b, tier.width, g),
                                                onmouseleave: scheduleHide,
                                            },
                                            t.span({ className: "block-label" }, g.label),
                                        );
                                    }),
                                ),
                            );
                        }),
                    ),
                ),
            );
        }));
    }

    async function load() {
        await loading(state, async () => {
            const res = await app.pb.send("/api/v1/frequency-overview", {
                method: "GET",
                query: { scope },
                requestKey: "frequency_overview",
            });
            render(res.devices || [], res.bands || []);
        });
    }

    const el = t.div(
        {
            className: () => `freq-overview ${state.isLoading ? "loading" : ""}`,
            onunmount: () => clearTimeout(hideTimer),
        },
        loader(state),
        t.div(
            { className: "freq-toolbar" },
            t.label({ className: "inline-select" }, t.span(null, "Devices"), select),
            t.div(
                { className: "freq-legend" },
                t.span({ className: "legend-item" }, t.span({ className: "swatch used" }), " In use"),
                t.span({ className: "legend-item" }, t.span({ className: "swatch available" }), " Available"),
                t.span({ className: "legend-item" }, t.span({ className: "swatch invalid" }), " Invalid"),
            ),
        ),
        bandsEl,
        tip,
    );

    return { el, load };
}
