package orion

import (
	"flag"
	"os"
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
	} else {
		bv := flag.Bool("verbose", false, "Enable verbose (console) logging")
		flag.Parse()
		verbose = bv
	}
}

func parseEnv() {
	registerToWatchdogByDefault = new(bool)
	wd := os.Getenv("WATCHDOG")
	*registerToWatchdogByDefault = wd == "true" || wd == "1"
}
