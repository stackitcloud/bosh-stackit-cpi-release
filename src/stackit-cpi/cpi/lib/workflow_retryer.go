package lib

import "fmt"

type Retryer struct {
	Workflow            func() error
	OnFail              func() error
	RetryAbleErrorCheck []func(e error) bool
	Logger              LoggerInterface
	Attempts            int
}

func (r *Retryer) Do() (err error) {
	var attempt int
outer:
	for attempt = range r.Attempts {
		r.Logger.Info("starting workflow", "attempt", fmt.Sprintf("%d/%d", attempt+1, r.Attempts))
		err = r.Workflow()
		if err != nil {
			r.Logger.Info("failed workflow attempt", "attempt", fmt.Sprintf("%d/%d", attempt+1, r.Attempts), "error", err)
			retry := false
			for _, continuable := range r.RetryAbleErrorCheck {
				if continuable(err) {
					r.Logger.Info("error is retryable")
					if cleanErr := r.OnFail(); cleanErr != nil {
						r.Logger.Debug("Will not retry because cleaning up between attempts failed", "error", cleanErr)
						break
					}
					retry = true
					break
				}
			}
			if retry {
				continue outer
			}
		}
		break
	}
	if err != nil {
		r.Logger.Debug("retryer failed terminally", "attempts", fmt.Sprintf("%d/%d", attempt+1, r.Attempts), "final_error", err)
	}
	return err
}
