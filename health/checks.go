package health

type Dependency struct {
	CheckIsWorking func() (string, *oerror.Error)
	Timeout        time.Duration
	Name           string
}

var (
	// The latest summary of errors happening when running health checks. 
	summary = []error{}
)

func GetSummaryOfHealthChecks() []error {
	// TODO: mutex for read
	return summary
}

func ResetSummaryOfHealthChecks() {
	// TODO: mutex for writing
	summary = []error{}
}

func AppendHealthCheckError(err error) {
	// TODO: mutex for writing
	summary = append(summary, err)
}