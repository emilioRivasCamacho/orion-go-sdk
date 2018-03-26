package orion

import (
	"flag"
	"strconv"
)

func ParseFlags() {
	f := flag.Lookup("verbose")
	if f == nil {
		bw := flag.Bool("watchdog", false, "Register to watchdog at init and periodically")
		bv := flag.Bool("verbose", false, "Enable verbose (console) logging")
		flag.Parse()
		registerToWatchdogByDefault = bw
		verbose = bv
	} else {
		b, _ := strconv.ParseBool(f.Value.String())
		verbose = &b
		b, _ = strconv.ParseBool(flag.Lookup("watchdog").Value.String())
		registerToWatchdogByDefault = &b
	}
}
