// OpenSOHO extension of the PocketBase superuser UI.
//
// The upstream UI concatenates this file into /_/extensions.js and runs it
// before the router starts. The other extension files are served from
// /_/extensions/opensoho/.

const base = app.pb.buildURL("/_/extensions/opensoho/");

document.head.appendChild(t.link({ rel: "stylesheet", href: base + "css/opensoho.css" }));

app.store.mainLogo = base + "images/logo.svg";
app.store.headerLogo = base + "images/logo.svg";
app.store.favicon = base + "images/favicon.png";

app.store.creditLinks = [
    {
        href: "https://opensoho.github.io/docs/",
        icon: "ri-book-open-line",
        label: "Docs",
    },
    {
        href: "https://github.com/rubenbe/opensoho/releases",
        icon: "ri-github-line",
        label: "OpenSOHO",
    },
];

document.addEventListener("mount:pageInstaller", () => {
    app.store.title = app.store.title.replace("PocketBase", "OpenSOHO");
});

app.opensoho = await import(base + "js/helpers.js");
