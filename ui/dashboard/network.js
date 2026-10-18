import { loader, loading } from "./charts.js";

// Remember the last-selected device across page refreshes.
const STORAGE_KEY = "network_overview_device";

const LLDP_WARNING = "⚠️ No LLDP data, verify whether lldpd is installed on this device.";

/**
 * Network overview card: the ports of one device, their link, PoE, bridge
 * and LLDP neighbours (see /api/v1/network-overview).
 *
 * @return {{el: Element, load: Function}}
 */
export function networkOverview() {
    const state = store({ isLoading: false });

    let scope = window.localStorage.getItem(STORAGE_KEY) || ""; // the device id currently shown

    const select = t.select({
        onchange: (e) => {
            scope = e.target.value;
            load();
        },
    });

    const statusEl = t.span({ className: "net-status" });
    const tableEl = t.div();

    // Jump the card to the neighbour device (stays on the dashboard).
    function openDevice(id) {
        scope = id;
        load();
    }

    function portRow(p, hasPoe) {
        let poe;
        if (p.poe) {
            poe = `⚡ ${p.poe.toFixed(1)}W`;
        } else if (p.poe != null) {
            poe = [t.span({ className: "net-poe-idle", title: "PoE port, not supplying power" }, "⚡"), " —"];
        } else {
            poe = "—";
        }

        return t.tr(
            null,
            t.td({ className: "net-port" }, p.port),
            p.speed
                ? t.td(null, t.span({ className: "net-dot net-dot--up" }), p.speed)
                : t.td({ className: "net-muted" }, t.span({ className: "net-dot" }), "no link"),
            hasPoe ? t.td({ className: "net-muted" }, poe) : null,
            t.td({ className: "net-muted" }, p.bridge || "—"),
            t.td(
                null,
                p.neighbors?.length
                    ? p.neighbors.map((n) => {
                        const label = `${n.name || "(unnamed)"} (${n.mac})`;
                        return t.div(
                            { className: "net-neighbour" },
                            n.knownDeviceId
                                ? t.button(
                                    { type: "button", className: "device-link", onclick: () => openDevice(n.knownDeviceId) },
                                    label,
                                )
                                : t.span(null, label),
                        );
                    })
                    : t.span({ className: "net-muted" }, "—"),
            ),
        );
    }

    function render(devices, ports) {
        select.replaceChildren(...devices.map((d) => t.option({ value: d.id }, d.name || d.id)));
        select.value = scope;

        const device = devices.find((d) => d.id === scope);
        const health = device?.health || "";
        const hasPoe = ports.some((p) => p.poe != null);
        // Any LLDP neighbour on any port means lldpd is reporting; none likely means
        // lldpd isn't installed/running on the device.
        const hasLldp = ports.some((p) => p.neighbors?.length);

        const status = [];
        if (health) {
            const variant = health === "healthy" ? "success" : health === "unhealthy" ? "danger" : "";
            status.push(t.span({ className: `label ${variant} data--health_status--${health}` }, health));
        }
        if (!hasLldp) {
            // link to the LuCI package manager of the device, where lldpd can be installed
            status.push(
                device?.ip
                    ? t.a(
                        {
                            className: "net-warning",
                            href: `http://${device.ip}/cgi-bin/luci/admin/system/package-manager`,
                            target: "_blank",
                            rel: "noopener noreferrer",
                            title: "Open the device's LuCI package manager",
                        },
                        LLDP_WARNING,
                    )
                    : t.span({ className: "net-warning" }, LLDP_WARNING),
            );
        }
        statusEl.replaceChildren(...status);

        if (!ports.length) {
            tableEl.replaceChildren(t.div({ className: "net-empty" }, "No ports known for this device."));
            return;
        }

        tableEl.replaceChildren(t.div(
            { className: "net-scroll" },
            t.table(
                { className: "net-table" },
                t.thead(
                    null,
                    t.tr(
                        null,
                        t.th(null, "Port"),
                        t.th(null, "Link"),
                        hasPoe ? t.th(null, "PoE") : null,
                        t.th(null, "Bridge"),
                        t.th(null, "Neighbours"),
                    ),
                ),
                t.tbody(null, ports.map((p) => portRow(p, hasPoe))),
            ),
        ));
    }

    async function load() {
        await loading(state, async () => {
            const res = await app.pb.send("/api/v1/network-overview", {
                method: "GET",
                query: { device: scope },
                requestKey: "network_overview",
            });
            // adopt the server-resolved scope so the selector syncs on first load
            if (res.scope) {
                scope = res.scope;
            }
            if (scope) {
                window.localStorage.setItem(STORAGE_KEY, scope);
            }
            render(res.devices || [], res.ports || []);
        });
    }

    const el = t.div(
        { className: () => `net-overview ${state.isLoading ? "loading" : ""}` },
        loader(state),
        t.div(
            { className: "net-toolbar" },
            t.label({ className: "inline-select" }, t.span(null, "Device"), select),
            statusEl,
        ),
        tableEl,
    );

    return { el, load };
}
