package jobs

type Reporter interface {
	Progress(Progress)
	Line(string)
}

type nopReporter struct{}

func (nopReporter) Progress(Progress) {}
func (nopReporter) Line(string)       {}

// NopReporter is a no-op Reporter.
var NopReporter Reporter = nopReporter{}
