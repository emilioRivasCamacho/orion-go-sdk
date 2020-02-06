package health

import (
	"fmt"
	"net/http"

	"github.com/gig/orion-go-sdk/env"

	"github.com/go-chi/chi"
)

/*
	TODO: Whenever we implement HTTP as a comm protocol for orion, we should remove this init function
	and call InstallHealthcheck instead.
*/

func init() {
	disableHealthChecks := env.Truthy("DISABLE_HEALTH_CHECK")

	if !disableHealthChecks {
		r := chi.NewRouter()
		InstallHealthcheck(r, "/healthcheck")

		go func() {
			defer func() {
				if r := recover(); r != nil {
					// TODO: What to do with the error?
					fmt.Println(r)
				}
			}()
			// TODO: Handle this error
			_ = http.ListenAndServe(":9001", r)
		}()
	}
}

func InstallHealthcheck(router chi.Router, endpointPath string) {
	router.Get(endpointPath, func(w http.ResponseWriter, r *http.Request) {
		summary := GetSummaryOfHealthChecks()

		if len(summary) == 0 {
			w.WriteHeader(200)
			// TODO: Handle this error
			_, _ = w.Write([]byte("OK"))
		} else {
			summaryString := "Error(s):\n"
			for _, err := range summary {
				summaryString = summaryString + err.Error() + "\n"
			}
			w.WriteHeader(500)
			// TODO: Handle this error
			_, _ = w.Write([]byte(summaryString))
		}
	})
}
