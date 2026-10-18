/**
 * Defines the sidebar grouping and display order for collections.
 *
 * - Groups appear in the order listed here.
 * - Collections within each group appear in the order listed.
 * - Collections not listed here are appended at the bottom, ungrouped.
 */
app.opensoho.collectionGroups = [
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
        items: ["bridges", "ethernet", "dhcp_reservations", "dhcp_leases", "lldp"],
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

const nameReplacements = {
    "ssh_keys": "SSH keys",
    "mac_address": "MAC address",
    "dhcp_reservations": "DHCP reservations",
    "dhcp_leases": "DHCP leases",
    "ip_address": "IP address",
    "wifi_client_psk": "Wifi client PSK",
    "wifi_ssids": "Wifi SSIDs",
};

const nameReplacementsPattern = new RegExp("^" + Object.keys(nameReplacements).join("|"));

/**
 * Returns a human friendly version of a collection or field name.
 *
 * @param  {string} value
 * @return {string}
 */
app.opensoho.cleanupName = function(value) {
    const cleaned = value.replace(nameReplacementsPattern, (matched) => nameReplacements[matched.toLowerCase()]);
    return cleaned.replace(/_/g, " ");
};

/**
 * Converts bytes into a readable string.
 *
 * @param  {number} size
 * @return {string}
 */
app.opensoho.formatBytes = function(size) {
    if (size === 0) return "0 B";
    const units = ["B", "KB", "MB", "GB", "TB", "PB"];
    const i = Math.floor(Math.log(size) / Math.log(1024));
    return (size / Math.pow(1024, i)).toFixed(2) + " " + units[i];
};

/**
 * Returns true when the user is not allowed to make new records.
 *
 * @param  {string} name
 * @return {boolean}
 */
app.opensoho.isCollectionReadOnly = function(name) {
    switch (name?.toLowerCase()) {
        case "devices":
        case "dhcp_leases":
        case "ethernet":
        case "interfaces":
        case "radios":
        case "radio_frequencies":
        case "radio_ht_modes":
        case "radio_tx_powers":
            return true;
        default:
            return false;
    }
};

/**
 * Returns true when the user is not allowed to edit the field.
 *
 * @param  {string} collectionName
 * @param  {string} fieldName
 * @return {boolean}
 */
app.opensoho.isCollectionFieldReadOnly = function(collectionName, fieldName) {
    switch (collectionName?.toLowerCase()) {
        // full read-only
        case "bridges":
        case "dhcp_leases":
        case "interfaces":
            return true;
        // partial read-write
        case "ethernet":
            switch (fieldName.toLowerCase()) {
                case "config":
                    return false;
                default:
                    return true;
            }
        case "clients":
            switch (fieldName.toLowerCase()) {
                case "mac_address":
                case "alias":
                    return false;
                default:
                    return true;
            }
        case "radios":
            switch (fieldName.toLowerCase()) {
                case "interface":
                case "mac_address":
                case "radio":
                    return true;
                default:
                    return false;
            }
        case "devices":
            switch (fieldName.toLowerCase()) {
                case "ip_address":
                case "os":
                case "uuid":
                case "system":
                case "key":
                case "backend":
                case "error_reason":
                case "health_status":
                case "config":
                case "model":
                case "hardware_id":
                case "last_seen":
                case "config_status":
                case "tags":
                    return true;
                default:
                    return false;
            }
        // full read-write
        default:
            return false;
    }
};

/**
 * Returns a stable color for a network (or VLAN) name.
 *
 * @param  {string} forName
 * @return {string}
 */
app.opensoho.getNetworkColor = function(forName) {
    if (forName == null) {
        return "#eeeeee";
    } else if (forName == "lan") {
        return "#90f090";
    } else if (forName == "wan") {
        return "#f09090";
    }

    const prng = new app.opensoho.PrngHelper();
    return prng.derive_color(forName);
};

/**
 * Returns the sidebar icon of a collection.
 *
 * @param  {string} name
 * @return {string} Empty string when there is no OpenSOHO specific icon.
 */
app.opensoho.getCollectionIcon = function(name) {
    switch (name?.toLowerCase()) {
        case "port_tagging":
            return "ri-price-tag-line";
        case "bridges":
            return "ri-exchange-box-line";
        case "clients":
            return "ri-device-line";
        case "dhcp_reservations":
            return "ri-door-lock-box-line";
        case "vlan":
            return "ri-stack-line";
        case "wifi_ssids":
            return "ri-home-wifi-line";
        case "dhcp_leases":
            return "ri-home-office-line";
        case "ethernet":
            return "ri-git-fork-line";
        case "interfaces":
            return "ri-base-station-line";
        case "lldp":
            return "ri-phone-find-line";
        case "poe":
            return "ri-flashlight-line";
        case "radios":
        case "radio_frequencies":
        case "radio_ht_modes":
        case "radio_tx_powers":
            return "ri-broadcast-line";
        case "leds":
            return "ri-lightbulb-line";
        case "client_steering":
            return "ri-steering-2-line";
        case "wifi_aps":
        case "devices":
            return "ri-router-line";
        case "connected_clients":
            return "ri-wifi-line";
        case "disconnected_clients":
            return "ri-signal-wifi-off-line";
        case "ssh_keys":
            return "ri-key-line";
        case "settings":
            return "ri-tools-line";
        case "wifi_client_psk":
            return "ri-shield-keyhole-line";
        default:
            return "";
    }
};
