// The devices shown on the resource history chart, without DOM, so it can be
// unit tested with node (see ui/tests).
//
// A selection is { devices: [id, ...], slots: { id: slot } }. The slot is the
// device's color, which it keeps while other devices are (de)selected. The
// functions update the selection in place.

// The lowest slot no selected device uses.
export function freeSlot(selection, max) {
    const used = new Set(selection.devices.map((id) => selection.slots[id]));
    for (let slot = 0; slot < max; slot++) {
        if (!used.has(slot)) return slot;
    }
    return 0;
}

// Selects or deselects one device; at most max devices are selected.
export function setDevice(selection, id, on, max) {
    const selected = selection.devices.includes(id);
    if (on === selected || (on && selection.devices.length >= max)) {
        return;
    }
    if (on) {
        selection.slots[id] = freeSlot(selection, max);
        selection.devices.push(id);
    } else {
        selection.devices = selection.devices.filter((d) => d !== id);
        delete selection.slots[id];
    }
}

// Replaces the selection with the first max of ids; the devices that stay
// selected keep their slot.
export function selectDevices(selection, ids, max) {
    const wanted = ids.slice(0, max);
    selection.devices = selection.devices.filter((id) => wanted.includes(id));
    for (const id of Object.keys(selection.slots)) {
        if (!selection.devices.includes(id)) {
            delete selection.slots[id];
        }
    }
    for (const id of wanted) {
        if (!selection.devices.includes(id)) {
            selection.slots[id] = freeSlot(selection, max);
            selection.devices.push(id);
        }
    }
}

export function healthyDeviceIds(devices) {
    return devices.filter((d) => d.health_status === "healthy").map((d) => d.id);
}

// The label of a "select all ..." action for count devices, of which only
// max can be selected.
export function selectAllLabel(label, count, max) {
    return count > max ? `First ${max} ${label.replace(/^All /, "")}` : label;
}
