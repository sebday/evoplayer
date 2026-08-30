package playback

// dispatch runs fn on the actor's single worker goroutine.
func (a *Actor) dispatch(fn func()) {
	done := make(chan struct{})
	a.cmdCh <- func() {
		fn()
		close(done)
	}
	<-done
}

// dispatchErr runs fn on the worker and returns its error.
func (a *Actor) dispatchErr(fn func() error) error {
	errCh := make(chan error, 1)
	a.cmdCh <- func() {
		errCh <- fn()
	}
	return <-errCh
}

func (a *Actor) workerLoop() {
	for fn := range a.cmdCh {
		fn()
	}
}
