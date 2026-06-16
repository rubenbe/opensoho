// Wi-Fi frequency / channel / band helpers.
//
// These mirror the server-side logic in opensoho.go (frequencyToBand,
// frequencyToChannel and the HT-mode / band rules) so the dashboard can reason
// about channels client-side without an extra round-trip.

// Bands we render an overview for, in display order.
export const BANDS = ["2.4", "5", "6"];

export const BAND_LABELS = {
    "2.4": "2.4 GHz",
    "5": "5 GHz",
    "6": "6 GHz",
    "60": "60 GHz",
};

// Build the helpers below from the canonical channel plan so the channel<->freq
// mapping stays consistent.
function channels24() {
    const list = [];
    for (let ch = 1; ch <= 13; ch++) {
        list.push({ channel: ch, frequency: 2407 + ch * 5 });
    }
    list.push({ channel: 14, frequency: 2484 });
    return list;
}

function channels5() {
    const chans = [
        36, 40, 44, 48, 52, 56, 60, 64,
        100, 104, 108, 112, 116, 120, 124, 128, 132, 136, 140, 144,
        149, 153, 157, 161, 165,
    ];
    return chans.map((ch) => ({ channel: ch, frequency: 5000 + ch * 5 }));
}

function channels6() {
    const list = [];
    // 6 GHz 20 MHz channels: 1, 5, 9, ... 233 (step 4), freq = 5950 + 5*ch.
    for (let ch = 1; ch <= 233; ch += 4) {
        list.push({ channel: ch, frequency: 5950 + ch * 5 });
    }
    return list;
}

// Standard 20 MHz channel plan per band: ordered array of { channel, frequency }.
export const STANDARD_CHANNELS = {
    "2.4": channels24(),
    "5": channels5(),
    "6": channels6(),
};

// frequencyToBand maps a frequency in MHz to a band string, matching opensoho.go.
export function frequencyToBand(frequency) {
    if (frequency >= 2400 && frequency <= 2500) return "2.4";
    if (frequency >= 5170 && frequency <= 5835) return "5";
    if (frequency >= 5925 && frequency <= 7125) return "6";
    if (frequency >= 57000 && frequency <= 71000) return "60";
    return "unknown";
}

// frequencyToChannel maps a frequency in MHz to a channel number, matching
// opensoho.go. Returns null when the frequency is outside the known ranges.
export function frequencyToChannel(freqMHz) {
    if (freqMHz >= 2412 && freqMHz <= 2484) {
        if (freqMHz === 2484) return 14;
        return (freqMHz - 2407) / 5;
    }
    if (freqMHz >= 5180 && freqMHz <= 5825) return (freqMHz - 5000) / 5;
    if (freqMHz >= 5955 && freqMHz <= 7115) return (freqMHz - 5950) / 5;
    return null;
}

// htmodeWidth returns the channel width (in MHz, as a number) for an htmode such
// as "HT20", "VHT40", "HE160". A blank htmode is treated as 20 MHz, the
// conservative Wi-Fi default. Returns null for an unrecognised value.
export function htmodeWidth(htmode) {
    if (!htmode) return 20;
    const m = String(htmode).match(/(\d+)$/);
    return m ? parseInt(m[1], 10) : null;
}

// Channel widths we draw a bonding tier for, per band. 2.4 GHz only supports
// 20/40 MHz; 5/6 GHz support the wider bonded widths.
export const BAND_WIDTHS = {
    "2.4": [20, 40],
    "5": [20, 40, 80, 160],
    "6": [20, 40, 80, 160],
};

// bondingGroups returns the channel-bonding groups for a band at a given width,
// in column order. Each group is:
//   { frequencies: number[], startIndex: number, span: number, complete: boolean }
// where startIndex is the index into STANDARD_CHANNELS[band] of the group's first
// channel and span === frequencies.length. complete is false for a trailing chunk
// that does not have the full `width / 20` contiguous channels (it cannot form
// that width).
//
// For 5/6 GHz, channels are grouped within frequency-contiguous runs (consecutive
// plan entries 20 MHz apart). This prevents bonding across band gaps such as the
// 5 GHz 64->100 jump and the 144->149 boundary, yielding the correct standard
// bonding groups. For 2.4 GHz, whose channels overlap at 5 MHz spacing, groups are
// simply consecutive adjacent-channel chunks (an accepted approximation).
export function bondingGroups(band, width) {
    const plan = STANDARD_CHANNELS[band] || [];
    const k = width / 20;
    if (k <= 1) {
        return plan.map((c, i) => ({
            frequencies: [c.frequency],
            startIndex: i,
            span: 1,
            complete: true,
        }));
    }

    // Split into contiguous runs of plan indices. 2.4 GHz is treated as one run
    // (overlapping channels); 5/6 GHz break runs where the frequency step != 20.
    const runs = [];
    let run = [];
    for (let i = 0; i < plan.length; i++) {
        const contiguous =
            run.length === 0 ||
            band === "2.4" ||
            plan[i].frequency - plan[i - 1].frequency === 20;
        if (!contiguous) {
            runs.push(run);
            run = [];
        }
        run.push(i);
    }
    if (run.length) runs.push(run);

    const groups = [];
    for (const indices of runs) {
        for (let off = 0; off < indices.length; off += k) {
            const chunk = indices.slice(off, off + k);
            groups.push({
                frequencies: chunk.map((i) => plan[i].frequency),
                startIndex: chunk[0],
                span: chunk.length,
                complete: chunk.length === k,
            });
        }
    }
    return groups;
}
