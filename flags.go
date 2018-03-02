package orion

import (
	"flag"
)

var (
	parsed = false
)

func ParseFlags() {
	if !parsed {
		bw := flag.Bool("watchdog", false, "Register to watchdog at init and periodically")
		bv := flag.Bool("verbose", false, "Enable verbose (console) logging")
		flag.Parse()
		registerToWatchdogByDefault = bw
		verbose = bv
	}
	parsed = true
}
