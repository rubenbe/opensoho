FROM docker.io/openwrt/rootfs:x86_64-openwrt-24.10
RUN mkdir /var/lock/ && opkg update; opkg install openwisp-config openwisp-monitoring
COPY openwisp openwisp-monitoring /etc/config/
CMD ["busybox", "sleep", "1000d"]
