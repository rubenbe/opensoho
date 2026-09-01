#!/bin/sh
# OpenWISP hotplug script, deployed to /etc/hotplug.d/openwisp/opensoho.
#
# /sbin/hotplug-call sources every script under /etc/hotplug.d/<subsystem>/
# with `. $script` (shell dot-sourcing) - it does not exec them, so shebang
# lines are ignored and the file's content is parsed as shell syntax
# directly. dump-radios.uc is ucode, not shell, so it can't be deployed here
# itself: hotplug-call's real sourcing path fails with a shell syntax error
# on it every cycle, even though invoking `ucode dump-radios.uc` directly (as
# used while developing/testing) works fine and masks the problem entirely -
# that mismatch is what caused issue #59's fix to appear to never take effect
# in production despite testing clean. This shim is real, sourceable shell
# that hands off to the actual ucode script as a child process instead.
exec ucode /usr/share/opensoho/dump-radios.uc "$@"
