package orion

import (
	"flag"
	"strconv"
)

// ParseFlags will parse the flags if they are not parsed yet
// If they are already parsed the func will lookup for the "--verbose" and "--watchdog"
func parseFlags() {
	if flag.Parsed() {
		v := flag.Lookup("verbose")
		if v != nil {
			b, _ := strconv.ParseBool(v.Value.String())
			verbose = &b
		}
		w := flag.Lookup("watchdog")
		if w != nil {
			b, _ := strconv.ParseBool(w.Value.String())
			registerToWatchdogByDefault = &b
		}
	} else {
		bw := flag.Bool("watchdog", false, "Register to watchdog at init and periodically")
		bv := flag.Bool("verbose", false, "Enable verbose (console) logging")
		flag.Parse()
		registerToWatchdogByDefault = bw
		verbose = bv
	}
}
