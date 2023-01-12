package test

import "time"

func RunWithTimeout(f func() error, timeout time.Duration, interval time.Duration) (isTimeout bool, err error) {
	control := make(chan bool)
	timeoutTimer := time.NewTimer(timeout)
	go func() {
		loop := true
		intervalTimer := time.NewTimer(interval)
		for loop {
			select {
			case <-control:
				return
			case <-intervalTimer.C:
				err = f()
				if err != nil {
					intervalTimer.Reset(interval)
				} else {
					loop = false
				}
			}
		}
		control <- true
	}()

	select {
	case <-control:
		return false, nil
	case <-timeoutTimer.C:
		control <- true
		return true, err
	}
}
