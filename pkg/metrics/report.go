package metrics

import "time"

type Reporter interface {
	Report(map[string]Metric)
}

func StartReporter(interval time.Duration, reporter Reporter) chan struct{} {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				reporter.Report(globalCollector.metrics)
			case <-stop:
				return
			}
		}
	}()
	return stop
}
