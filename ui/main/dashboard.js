// The dashboard page itself (ui/dashboard) is only loaded when opened.

document.head.appendChild(t.link({ rel: "stylesheet", href: app.opensoho.baseURL + "css/dashboard.css" }));

app.store.headerLinks.unshift({
    href: "#/dashboard",
    icon: "ri-dashboard-line",
    label: "Dashboard",
});

app.routes.superuserOnly("#/dashboard", async (route) => {
    const { pageDashboard } = await import(app.opensoho.baseURL + "dashboard/page.js");
    return pageDashboard(route);
});
