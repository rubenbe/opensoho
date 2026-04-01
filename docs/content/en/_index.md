---
title: "OpenSOHO"
linkTitle: "OpenSOHO"
---

{{< blocks/cover title="OpenSOHO" image_anchor="top" height="min" color="primary" >}}
<a class="btn btn-lg btn-primary me-3 mb-4" href="/docs/">
  Get Started <i class="fas fa-arrow-alt-circle-right ms-2"></i>
</a>
<a class="btn btn-lg btn-secondary me-3 mb-4" href="https://github.com/rubenbe/opensoho/releases">
  <i class="fab fa-github me-2"></i> Releases
</a>
<p class="lead mt-5">Manage your Small Office/Home Office OpenWRT network. Similar as other cloud controllers from Ubiquity, Omada, etc</p>
{{< /blocks/cover >}}

{{< blocks/lead color="dark" >}}
OpenSOHO manages 2 to 20 OpenWRT devices from a single web interface.
Configure Wifi, VLANs, radios, and LEDs — and monitor device health in real time.
Compatible with <strong>openwisp-config</strong> and <strong>openwisp-monitoring</strong>.
{{< /blocks/lead >}}

{{< blocks/section color="white" >}}
<div class="col-lg-6 mb-5 mb-lg-0">
  <h2>Everything in one place</h2>
  <p class="mt-3">OpenSOHO gives you a unified view of all your network devices. Push the same Wifi configuration to every access point, apply VLANs across your whole network, or fine-tune individual devices — all without touching each router separately.</p>
  <ul class="mt-3">
    <li>Wifi SSID and encryption management</li>
    <li>VLAN configuration across all devices</li>
    <li>Radio frequency tuning</li>
    <li>LED control</li>
    <li>Real-time device health monitoring</li>
    <li>Connected client overview</li>
    <li>Home Assistant integration</li>
  </ul>
  <p class="mt-4">Tested on <strong>OpenWRT 24.10.x</strong> and <strong>OpenWRT 25.12.x</strong> (DSA).</p>
</div>
<div class="col-lg-6 text-center">
  <a href="https://raw.githubusercontent.com/opensoho/assets/074a4c5c353fcbb295e2da84fb3490869c6c4de0/devices.png" target="_blank">
    <img src="https://raw.githubusercontent.com/opensoho/assets/074a4c5c353fcbb295e2da84fb3490869c6c4de0/devices.png"
         alt="OpenSOHO device overview screenshot"
         class="img-fluid rounded shadow" />
  </a>
</div>
{{< /blocks/section >}}

{{< blocks/section color="light" type="row" >}}
{{% blocks/feature icon="fa-network-wired" title="Simple Network Management" %}}
Manage 2 to 20 OpenWRT devices from a single interface. Configure Wifi SSIDs, VLANs, radios, and LEDs across your entire network at once.
{{% /blocks/feature %}}

{{% blocks/feature icon="fab fa-github" title="Open Source" url="https://github.com/rubenbe/opensoho" %}}
Fully open source on GitHub. Compatible with `openwisp-config` and `openwisp-monitoring`. Contributions and feedback welcome!
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-home" title="SOHO Focused" %}}
Inspired by OpenWISP but designed for networks too small to need it. No templates, no multi-tenancy — just the features you actually need.
{{% /blocks/feature %}}
{{< /blocks/section >}}

{{< blocks/section color="white" type="row" >}}
{{% blocks/feature icon="fab fa-github" title="Source Code" url="https://github.com/rubenbe/opensoho" %}}
Browse the source, open issues, or submit a pull request on GitHub.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-download" title="Releases" url="https://github.com/rubenbe/opensoho/releases" %}}
Download the latest release or pull a prebuilt container image from the [GitHub Container Registry](https://github.com/orgs/opensoho/packages).
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-mug-hot" title="Support the Project" url="https://www.buymeacoffee.com/rubenbe" %}}
If OpenSOHO saves you time, consider supporting the project!
{{% /blocks/feature %}}
{{< /blocks/section >}}
