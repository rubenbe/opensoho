// Records list customizations: field rendering, dates, column names,
// read-only collections and the auto refresh.

// -------------------------------------------------------------------
// Field values
// -------------------------------------------------------------------

/**
 * Replaces the view (records list, preview) of a field type.
 * The custom view returns undefined to fall back to the original one.
 */
function overrideView(type, customView) {
    const original = app.fieldTypes[type].view;
    app.fieldTypes[type].view = (props) => customView(props) || original(props);
}

function missingValue() {
    return t.span({ className: "missing-value" });
}

overrideView("text", (props) => {
    // the VLAN name gets its network color
    if (props.record?.collectionName == "vlan" && props.field.name == "name") {
        return t.div({ className: "record-field-view field-type-text" }, () => {
            const value = props.record[props.field.name];
            if (!value) {
                return missingValue();
            }

            return t.span(
                {
                    className: "label",
                    style: `background-color: ${app.opensoho.getNetworkColor(value)}; color: #1a1a24;`,
                },
                t.span({ className: "txt txt-ellipsis" }, value),
            );
        });
    }

    // IP addresses link to the device
    if (props.short && props.field.name == "ip_address") {
        return t.div({ className: "record-field-view field-type-text" }, () => {
            const value = props.record[props.field.name];
            if (!value) {
                return missingValue();
            }

            return t.a(
                {
                    className: "txt-ellipsis",
                    href: `http://${value}`,
                    target: "_blank",
                    rel: "noopener noreferrer",
                    ariaDescription: app.attrs.tooltip("Open in new tab"),
                    onclick: (e) => e.stopPropagation(),
                },
                app.utils.truncate(value),
            );
        });
    }
});

overrideView("number", (props) => {
    if (props.field.name != "tx_bytes" && props.field.name != "rx_bytes") {
        return;
    }

    return t.div({ className: "record-field-view field-type-number" }, () => {
        const value = props.record[props.field.name];
        if (value == null) {
            return missingValue();
        }

        return t.span(
            { className: "txt", ariaDescription: app.attrs.tooltip(`${value} bytes`) },
            app.opensoho.formatBytes(value),
        );
    });
});

// select values get a data--{field}--{value} class, used to color the
// device health and config status (see css/opensoho.css)
overrideView("select", (props) => {
    return t.div(
        { className: "record-field-view field-type-select" },
        () => {
            const opts = app.utils.toArray(props.record[props.field.name], false);

            if (!opts.length) {
                return missingValue();
            }

            return opts.map((opt) => {
                return t.span({
                    className: `label data--${props.field.name}--${opt}`,
                    title: opt,
                    textContent: app.utils.truncate(opt, 100),
                });
            });
        },
    );
});

// relations to a VLAN get its network color
const originalRecordSummary = app.components.recordSummary;
app.components.recordSummary = function(record, meta = null) {
    const el = originalRecordSummary(record, meta);

    if (record?.collectionName == "vlan" && record.name) {
        el.style.backgroundColor = app.opensoho.getNetworkColor(record.name);
        el.style.color = "#1a1a24";
    }

    return el;
};

// show the local time, with the UTC time in the tooltip
app.components.formattedDate = function(propsArg = {}) {
    const props = store({
        rid: undefined,
        id: undefined,
        hidden: undefined,
        value: "",
        short: false,
    });

    const watchers = app.utils.extendStore(props, propsArg);

    return t.div(
        {
            rid: props.rid,
            id: () => props.id,
            hidden: () => props.hidden,
            ariaDescription: app.attrs.tooltip(() => {
                if (!props.value) {
                    return null;
                }

                return props.value.replace(/Z$/, "") + " UTC";
            }),
            "html-class": "formatted-date",
            className: () => `formatted-date ${props.short ? "short" : "full"}`,
            onunmount: () => {
                watchers.forEach((w) => w?.unwatch());
            },
        },
        () => {
            if (!props.value) {
                return missingValue();
            }

            const parts = app.utils.toLocalDatetime(props.value).split(" ");

            return [
                t.span({ className: "primary-date" }, parts[0]),
                t.span({ className: "secondary-date" }, parts[1]),
            ];
        },
    );
};

// files without preview are opened in a new tab instead of downloaded,
// and middle clicking any file opens it in a new tab
const originalRecordFileThumb = app.components.recordFileThumb;
app.components.recordFileThumb = function(propsArg = {}) {
    const el = originalRecordFileThumb(propsArg);

    async function openInNewTab() {
        const record = typeof propsArg.record == "function" ? propsArg.record() : propsArg.record;
        const filename = typeof propsArg.filename == "function" ? propsArg.filename() : propsArg.filename;
        try {
            const token = await app.getFileToken(record.collectionId);
            window.open(app.pb.files.getURL(record, filename, { token }), "_blank", "noreferrer,noopener");
        } catch (err) {
            console.warn("openInNewTab file token failure:", err);
        }
    }

    el.addEventListener("auxclick", (e) => {
        e.preventDefault();
        e.stopPropagation();
        openInNewTab();
    });

    // runs before the upstream onclick handler
    el.addEventListener("click", (e) => {
        if (el.title.startsWith("Download")) {
            e.stopImmediatePropagation();
            e.preventDefault();
            openInNewTab();
        }
    }, { capture: true });

    return el;
};

// -------------------------------------------------------------------
// Collections page
// -------------------------------------------------------------------

// open the devices when no collection was selected before
if (!window.localStorage.getItem("pbLastActiveCollection")) {
    window.localStorage.setItem("pbLastActiveCollection", "devices");
}

/**
 * Keeps the text of the matching elements cleaned up with app.opensoho.cleanupName.
 *
 * @param  {Element} root
 * @param  {string}  selector
 * @return {MutationObserver}
 */
function cleanupNames(root, selector) {
    function apply() {
        for (const el of root.querySelectorAll(selector)) {
            const cleaned = app.opensoho.cleanupName(el.textContent);
            if (el.textContent != cleaned) {
                el.textContent = cleaned;
            }
        }
    }

    apply();

    const observer = new MutationObserver(apply);
    observer.observe(root, { childList: true, subtree: true, characterData: true });

    return observer;
}

document.addEventListener("mount:pageCollections", (e) => {
    const page = e.detail;

    // read-only collections can't have new records (see css/opensoho.css)
    const readOnlyWatcher = watch(() => app.store.activeCollection?.name, (name) => {
        page.toggleAttribute("data-opensoho-readonly", app.opensoho.isCollectionReadOnly(name));
    });

    const breadcrumbs = page.querySelector(".page-header .breadcrumbs");
    const breadcrumbsObserver = breadcrumbs && cleanupNames(breadcrumbs, ":scope > div:not(:first-child)");

    document.addEventListener("unmount:pageCollections", function onunmount(e) {
        if (e.detail == page) {
            readOnlyWatcher.unwatch();
            breadcrumbsObserver?.disconnect();
            document.removeEventListener("unmount:pageCollections", onunmount);
        }
    });
});

document.addEventListener("mount:recordsListTable", (e) => {
    const table = e.detail;
    const observer = cleanupNames(table, "th.sort-handle .txt");

    document.addEventListener("unmount:recordsListTable", function onunmount(e) {
        if (e.detail == table) {
            observer.disconnect();
            document.removeEventListener("unmount:recordsListTable", onunmount);
        }
    });
});

// -------------------------------------------------------------------
// Auto refresh
// -------------------------------------------------------------------
//
// Every 15s the records shown in the list are loaded again with the same
// query as the list itself. Changed and deleted records are passed to the
// list with the record:save and record:delete events, which updates them in
// place (keeping the scroll position). New records are only added when the
// list is scrolled to the top, as the list reloads itself for them.

const AUTO_REFRESH_INTERVAL = 15000;

// the queries of the records list and the records it received
const listState = {
    collection: "",
    options: null,
    pages: 0,
    records: new Map(), // id => JSON
};

function snapshot(record) {
    return JSON.stringify(record);
}

const originalCollection = app.pb.collection;
app.pb.collection = function(idOrName) {
    const service = originalCollection.call(this, idOrName);

    if (!service.__opensohoGetList) {
        service.__opensohoGetList = true;

        const originalGetList = service.getList;
        service.getList = function(page, perPage, options = {}) {
            const result = originalGetList.call(service, page, perPage, options);

            if (!options.requestKey?.startsWith("records_list_")) {
                return result;
            }

            return result.then((list) => {
                if (page == 1) {
                    listState.collection = idOrName;
                    listState.options = { ...options, requestKey: "opensoho_auto_refresh" };
                    listState.perPage = perPage;
                    listState.records = new Map();
                }
                listState.pages = page;
                for (const record of list.items) {
                    listState.records.set(record.id, snapshot(record));
                }
                return list;
            });
        };
    }

    return service;
};

async function autoRefresh(page) {
    const list = page.querySelector('[data-pb="recordsList"]');
    if (
        document.hidden
        || !listState.options
        || listState.collection != app.store.activeCollection?.name
    ) {
        return;
    }

    const collection = listState.collection;
    const expected = listState.records;
    const refreshed = new Map();

    try {
        for (let i = 1; i <= listState.pages; i++) {
            const result = await originalCollection.call(app.pb, collection).getList(
                i,
                listState.perPage,
                listState.options,
            );
            for (const record of result.items) {
                refreshed.set(record.id, record);
            }
            if (result.items.length < listState.perPage) {
                break;
            }
        }
    } catch (err) {
        if (!err?.isAbort) {
            console.warn("auto refresh failed:", err);
        }
        return;
    }

    // the list was reloaded in the meantime
    if (listState.records !== expected) {
        return;
    }

    const isAtTop = !list || list.scrollTop == 0;

    for (const [id, record] of refreshed) {
        const json = snapshot(record);
        const old = expected.get(id);
        if (old == json || (old === undefined && !isAtTop)) {
            continue;
        }
        if (old !== undefined) {
            expected.set(id, json);
        } else {
            // updates the total count
            document.dispatchEvent(new CustomEvent("record:create", { detail: record }));
        }
        document.dispatchEvent(new CustomEvent("record:save", { detail: record }));
    }

    for (const id of [...expected.keys()]) {
        if (!refreshed.has(id)) {
            expected.delete(id);
            document.dispatchEvent(new CustomEvent("record:delete", {
                detail: { id, collectionId: collection, collectionName: collection },
            }));
        }
    }
}

document.addEventListener("mount:pageCollections", (e) => {
    const page = e.detail;
    const intervalId = setInterval(() => autoRefresh(page), AUTO_REFRESH_INTERVAL);

    document.addEventListener("unmount:pageCollections", function onunmount(e) {
        if (e.detail == page) {
            clearInterval(intervalId);
            document.removeEventListener("unmount:pageCollections", onunmount);
        }
    });
});
