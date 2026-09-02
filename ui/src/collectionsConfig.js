/**
 * Defines the sidebar grouping and display order for collections.
 *
 * - Groups appear in the order listed here.
 * - Collections within each group appear in the order listed.
 * - Collections not listed here are appended at the bottom, ungrouped.
 */
export default [
    {
        name: "Devices",
        items: ["devices"],
    },
    {
        name: "Clients",
        items: ["connected_clients", "disconnected_clients", "clients"],
    },
    {
        name: "WiFi",
        items: ["wifi_ssids", "wifi_aps", "radios", "wifi_client_psk", "interfaces"],
    },
    {
        name: "Network",
        items: ["bridges", "ethernet", "dhcp_leases", "lldp"],
    },
    {
        name: "VLAN",
        items: ["vlan", "port_tagging"],
    },
    {
        name: "System",
        items: ["ssh_keys", "client_steering", "leds", "settings", "poe", "api_users"],
    },
    {
        name: "Capabilities",
        items: ["radio_frequencies", "radio_ht_modes", "radio_tx_powers"],
    },
];
