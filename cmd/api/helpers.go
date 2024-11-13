package main

import "fmt"

func (app *application) startBackgroundJob(job func()) {
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		defer func() {
			if err := recover(); err != nil {
				app.logger.Info("Background Job", "Err", fmt.Errorf("%s", err))
			}
		}()
		job()
	}()
}
