// OpenSOHO extension of the PocketBase superuser UI.
//
// The files in ui/main are concatenated into the main.js of the extension
// (see ui/embed.go), which the upstream UI runs before starting its router.
// Each file runs in its own block scope; they share code via app.opensoho.

app.opensoho = {
    // the extension files are served from here
    baseURL: app.pb.buildURL("/_/extensions/opensoho/"),
};

document.head.appendChild(t.link({ rel: "stylesheet", href: app.opensoho.baseURL + "css/opensoho.css" }));

app.store.mainLogo = app.opensoho.baseURL + "images/logo.svg";
app.store.headerLogo = app.opensoho.baseURL + "images/logo.svg";
app.store.favicon = app.opensoho.baseURL + "images/favicon.png";

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
