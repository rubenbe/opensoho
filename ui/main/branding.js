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

// there is no mail setup to reset a password, link to the FAQ instead
document.addEventListener("mount:authWithPasswordForm", (e) => {
    function replaceLink() {
        const link = e.detail.querySelector('a[href="#/request-password-reset"]');
        if (link) {
            link.href = "https://opensoho.github.io/docs/faq/#i-forgot-my-admin-password--how-do-i-reset-it";
            link.target = "_blank";
            link.rel = "noopener noreferrer";
        }
    }

    // the form content can be rendered after the form is mounted
    replaceLink();
    setTimeout(replaceLink, 0);
});
