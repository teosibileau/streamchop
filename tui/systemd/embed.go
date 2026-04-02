package systemd

import _ "embed"

//go:embed streamchop.service.template
var ServiceTemplate string

//go:embed watchdog.sh
var WatchdogScript []byte
