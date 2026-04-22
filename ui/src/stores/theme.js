import { writable } from "svelte/store";

const STORAGE_KEY = "theme";

function applyTheme(value) {
    if (value === "auto") {
        document.documentElement.removeAttribute("data-theme");
    } else {
        document.documentElement.setAttribute("data-theme", value);
    }
}

function createThemeStore() {
    const stored = localStorage.getItem(STORAGE_KEY) || "auto";
    const { subscribe, set } = writable(stored);

    applyTheme(stored);

    return {
        subscribe,
        set(value) {
            localStorage.setItem(STORAGE_KEY, value);
            applyTheme(value);
            set(value);
        },
    };
}

export const theme = createThemeStore();
