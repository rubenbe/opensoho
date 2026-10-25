// Unit tests of ui/dashboard/selection.js, run with: node --test 'ui/tests/**/*.test.mjs'

import assert from "node:assert/strict";
import { describe, test } from "node:test";

import { freeSlot, healthyDeviceIds, selectAllLabel, selectDevices, setDevice } from "../dashboard/selection.js";

const MAX = 8;

function selection(devices = [], slots = {}) {
    return { devices: [...devices], slots: { ...slots } };
}

// n devices named d0, d1, ...; the ones in healthy are healthy, the rest not
function deviceList(n, healthy = []) {
    return Array.from({ length: n }, (_, i) => ({
        id: `d${i}`,
        name: `Device ${i}`,
        health_status: healthy.includes(i) ? "healthy" : "unhealthy",
    }));
}

describe("healthyDeviceIds", () => {
    test("only the healthy devices, in list order", () => {
        const devices = [
            { id: "a", health_status: "healthy" },
            { id: "b", health_status: "unhealthy" },
            { id: "c", health_status: "" },
            { id: "d", health_status: "healthy" },
        ];
        assert.deepEqual(healthyDeviceIds(devices), ["a", "d"]);
    });

    test("no healthy devices", () => {
        assert.deepEqual(healthyDeviceIds(deviceList(3)), []);
        assert.deepEqual(healthyDeviceIds([]), []);
    });
});

describe("selectAllLabel", () => {
    test("unchanged when every device fits", () => {
        assert.equal(selectAllLabel("All devices", 3, MAX), "All devices");
        assert.equal(selectAllLabel("All devices", 8, MAX), "All devices");
        assert.equal(selectAllLabel("All healthy devices", 0, MAX), "All healthy devices");
    });

    test("names the limit when there are more devices", () => {
        assert.equal(selectAllLabel("All devices", 9, MAX), "First 8 devices");
        assert.equal(selectAllLabel("All healthy devices", 12, MAX), "First 8 healthy devices");
    });
});

describe("selectDevices", () => {
    test("all devices from an empty selection", () => {
        const s = selection();
        selectDevices(s, ["a", "b", "c"], MAX);
        assert.deepEqual(s, selection(["a", "b", "c"], { a: 0, b: 1, c: 2 }));
    });

    test("all healthy devices drops the unhealthy ones", () => {
        const devices = deviceList(4, [0, 2]);
        const s = selection();
        selectDevices(s, devices.map((d) => d.id), MAX);
        selectDevices(s, healthyDeviceIds(devices), MAX);
        assert.deepEqual(s, selection(["d0", "d2"], { d0: 0, d2: 2 }));
    });

    test("devices that stay selected keep their color", () => {
        const s = selection(["b", "c"], { b: 5, c: 1 });
        selectDevices(s, ["a", "b", "c", "d"], MAX);
        assert.equal(s.slots.b, 5);
        assert.equal(s.slots.c, 1);
        // the new devices take the lowest free slots
        assert.equal(s.slots.a, 0);
        assert.equal(s.slots.d, 2);
    });

    test("at most max devices, the first ones", () => {
        const ids = deviceList(11).map((d) => d.id);
        const s = selection();
        selectDevices(s, ids, MAX);
        assert.deepEqual(s.devices, ids.slice(0, 8));
        assert.deepEqual(Object.values(s.slots).sort(), [0, 1, 2, 3, 4, 5, 6, 7]);
    });

    test("every selected device has its own slot", () => {
        const s = selection(["x", "y"], { x: 0, y: 7 });
        selectDevices(s, deviceList(10).map((d) => d.id), MAX);
        assert.equal(new Set(Object.values(s.slots)).size, s.devices.length);
        assert.deepEqual(Object.keys(s.slots).sort(), [...s.devices].sort());
    });

    test("nothing to select clears the selection", () => {
        const s = selection(["a"], { a: 3 });
        selectDevices(s, [], MAX);
        assert.deepEqual(s, selection());
    });

    test("the default selection of a new dashboard", () => {
        const s = selection();
        selectDevices(s, deviceList(10).map((d) => d.id), MAX);
        assert.deepEqual(s.devices, ["d0", "d1", "d2", "d3", "d4", "d5", "d6", "d7"]);
        s.devices.forEach((id, i) => assert.equal(s.slots[id], i));
    });
});

describe("setDevice", () => {
    test("select takes the lowest free slot", () => {
        const s = selection(["a", "c"], { a: 0, c: 2 });
        setDevice(s, "b", true, MAX);
        assert.deepEqual(s, selection(["a", "c", "b"], { a: 0, c: 2, b: 1 }));
    });

    test("deselect frees the slot, the others keep theirs", () => {
        const s = selection(["a", "b", "c"], { a: 0, b: 1, c: 2 });
        setDevice(s, "a", false, MAX);
        assert.deepEqual(s, selection(["b", "c"], { b: 1, c: 2 }));
    });

    test("no ninth device", () => {
        const s = selection();
        selectDevices(s, deviceList(8).map((d) => d.id), MAX);
        const before = structuredClone(s);
        setDevice(s, "extra", true, MAX);
        assert.deepEqual(s, before);
    });

    test("selecting a selected device changes nothing", () => {
        const s = selection(["a"], { a: 4 });
        setDevice(s, "a", true, MAX);
        assert.deepEqual(s, selection(["a"], { a: 4 }));
    });

    test("deselecting an unselected device changes nothing", () => {
        const s = selection(["a"], { a: 4 });
        setDevice(s, "b", false, MAX);
        assert.deepEqual(s, selection(["a"], { a: 4 }));
    });
});

describe("freeSlot", () => {
    test("the lowest unused slot", () => {
        assert.equal(freeSlot(selection(), MAX), 0);
        assert.equal(freeSlot(selection(["a", "b"], { a: 0, b: 2 }), MAX), 1);
    });

    test("ignores slots of devices that are no longer selected", () => {
        assert.equal(freeSlot(selection(["b"], { a: 0, b: 1 }), MAX), 0);
    });
});

test("toggling devices and selecting all keeps every color stable", () => {
    const devices = deviceList(5, [1, 3]);
    const s = selection();
    selectDevices(s, devices.map((d) => d.id), MAX);
    const colors = { ...s.slots };

    setDevice(s, "d0", false, MAX);
    selectDevices(s, healthyDeviceIds(devices), MAX);
    setDevice(s, "d4", true, MAX);
    for (const id of s.devices) {
        if (id !== "d4") {
            assert.equal(s.slots[id], colors[id], id);
        }
    }
    // d4 had lost its slot when it was deselected, it gets the lowest free one
    assert.equal(s.slots.d4, 0);
});
