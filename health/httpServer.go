package health

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
)

/*
	TODO: Whenever we implement HTTP as a comm protocol for orion, we should remove this init function
	and call InstallHealthcheck instead.
*/

func init() {
	r := chi.NewRouter()
	InstallHealthcheck(r, "/healthcheck")

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// TODO: What to do with the error?
				fmt.Println(r)
			}
		}()
		http.ListenAndServe(":8080", r)
	}()
}

func InstallHealthcheck(router chi.Router, endpointPath string) {
	summary := GetSummaryOfHealthChecks()

	router.Get(endpointPath, func(w http.ResponseWriter, r *http.Request) {
		if len(summary) == 0 {
			w.WriteHeader(200)
			w.Write([]byte("OK"))
		} else {
			summaryString := "Error(s):\n"
			for _, err := range summary {
				summaryString = summaryString + err.Error() + "\n"
			}
			w.WriteHeader(500)
			w.Write([]byte(summaryString))
		}
	})
}
