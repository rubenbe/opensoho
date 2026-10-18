// Groups the collections in the sidebar (see app.opensoho.collectionGroups),
// with OpenSOHO icons and display names.
//
// The upstream sidebar has no API for this, so its pinned and regular
// collection groups are hidden (see css/opensoho.css) and replaced by our own
// groups. These mirror the collections that upstream would show, so its
// search keeps working.

function collectionItem(collection) {
    return t.button(
        {
            "html-data-collection-id": collection.id,
            type: "button",
            className: () =>
                `nav-item responsive-close ${collection.id == app.store.activeCollection?.id ? "active" : ""}`,
            title: collection.name,
            onauxclick: (e) => {
                e.preventDefault();
                window.open(`#/collections?collection=${collection.name}`, "_blank", "noreferrer,noopener");
            },
            onclick: (e) => {
                e.preventDefault();
                app.store.activeCollection = collection.name;
            },
        },
        t.i({
            className: app.opensoho.getCollectionIcon(collection.name)
                || app.collectionTypes[collection.type]?.icon
                || app.utils.fallbackCollectionIcon,
            ariaHidden: true,
        }),
        t.span({ className: "txt" }, app.opensoho.cleanupName(collection.name)),
    );
}

/**
 * Groups and orders collections according to app.opensoho.collectionGroups.
 * Collections not listed there are appended in an "Other" group.
 *
 * @param  {Array} collections
 * @return {Array} [{ name: string, items: Array }]
 */
function groupCollections(collections) {
    const byName = Object.fromEntries(collections.map((c) => [c.name, c]));
    const used = new Set();

    const groups = app.opensoho.collectionGroups
        .map(({ name, items }) => ({
            name,
            items: items.map((n) => byName[n]).filter(Boolean),
        }))
        .filter(({ items }) => items.length > 0);

    for (const { items } of groups) {
        for (const c of items) used.add(c.name);
    }

    const rest = collections.filter((c) => !used.has(c.name));
    if (rest.length) {
        groups.push({ name: "Other", items: rest });
    }

    return groups;
}

function collectionGroups(list) {
    // the ids of the collections that the hidden upstream groups show
    const data = store({ ids: [] });

    function syncIds() {
        const ids = [...list.querySelectorAll(
            ".nav-group-pinned-collections [data-collection-id], .nav-group-regular-collections [data-collection-id]",
        )].map((el) => el.dataset.collectionId);

        if (ids.join(",") != data.ids.join(",")) {
            data.ids = ids;
        }
    }

    const observer = new MutationObserver(syncIds);

    return t.div(
        {
            className: "opensoho-collection-groups",
            onmount: () => {
                syncIds();
                observer.observe(list, { childList: true, subtree: true });
            },
            onunmount: () => observer.disconnect(),
        },
        () => {
            const collections = data.ids
                .map((id) => app.store.collections.find((c) => c.id == id))
                .filter(Boolean);

            return groupCollections(collections).map((group) => {
                return t.details(
                    { className: "nav-group", open: true },
                    t.summary(null, group.name),
                    group.items.map(collectionItem),
                );
            });
        },
    );
}

document.addEventListener("mount:pageSidebar", (e) => {
    const sidebar = e.detail;

    // the sidebar content is rendered after the sidebar itself is mounted,
    // not always with a mutation that the observer below can see
    function attach() {
        const list = sidebar.querySelector(".collections-list");
        if (list && !list.querySelector(":scope > .opensoho-collection-groups")) {
            list.prepend(collectionGroups(list));
        }
    }

    attach();
    setTimeout(attach, 0);

    const observer = new MutationObserver(attach);
    observer.observe(sidebar, { childList: true });
    document.addEventListener("unmount:pageSidebar", function onunmount(e) {
        if (e.detail == sidebar) {
            observer.disconnect();
            document.removeEventListener("unmount:pageSidebar", onunmount);
        }
    });
});
