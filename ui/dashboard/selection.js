// The items (devices, clients) shown on a history chart, without DOM, so it
// can be unit tested with node (see ui/tests).
//
// A selection is { selected: [id, ...], slots: { id: slot } }. The slot is
// the item's color, which it keeps while other items are (de)selected. The
// functions update the selection in place.

// The lowest slot no selected item uses.
export function freeSlot(selection, max) {
    const used = new Set(selection.selected.map((id) => selection.slots[id]));
    for (let slot = 0; slot < max; slot++) {
        if (!used.has(slot)) return slot;
    }
    return 0;
}

// Selects or deselects one item; at most max items are selected.
export function setItem(selection, id, on, max) {
    const selected = selection.selected.includes(id);
    if (on === selected || (on && selection.selected.length >= max)) {
        return;
    }
    if (on) {
        selection.slots[id] = freeSlot(selection, max);
        selection.selected.push(id);
    } else {
        selection.selected = selection.selected.filter((d) => d !== id);
        delete selection.slots[id];
    }
}

// Replaces the selection with the first max of ids; the items that stay
// selected keep their slot.
export function selectItems(selection, ids, max) {
    const wanted = ids.slice(0, max);
    selection.selected = selection.selected.filter((id) => wanted.includes(id));
    for (const id of Object.keys(selection.slots)) {
        if (!selection.selected.includes(id)) {
            delete selection.slots[id];
        }
    }
    for (const id of wanted) {
        if (!selection.selected.includes(id)) {
            selection.slots[id] = freeSlot(selection, max);
            selection.selected.push(id);
        }
    }
}

export function healthyDeviceIds(devices) {
    return devices.filter((d) => d.health_status === "healthy").map((d) => d.id);
}

// The label of a "select all ..." action for count items, of which only max
// can be selected.
export function selectAllLabel(label, count, max) {
    return count > max ? `First ${max} ${label.replace(/^All /, "")}` : label;
}

// The warning when one more item is picked than the max that can be shown;
// items is their plural name, e.g. "Devices".
export function limitWarning(items, max) {
    return `At most ${max} ${items.toLowerCase()} can be shown at once. Deselect one first.`;
}

// The ids of the n clients with the most traffic (download + upload), most
// first; clients without traffic are left out.
export function topByTraffic(clients, n) {
    return clients
        .filter((c) => c.download + c.upload > 0)
        .sort((a, b) => (b.download + b.upload) - (a.download + a.upload))
        .slice(0, n)
        .map((c) => c.id);
}
