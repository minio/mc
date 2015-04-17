package main

import (
	"io"
	"net"
	"time"

	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
)

// isValidRetry - check if we should retry for the given error sequence
func isValidRetry(err error) bool {
	if err == nil {
		return false
	}
	// handle io Errors separately since they are typed as *errors.String
	switch iodine.ToError(err) {
	case io.ErrUnexpectedEOF:
		return true
	}
	// DNSError, Network Operation error
	switch e := iodine.ToError(err).(type) {
	case *net.DNSError:
		return true
	case *net.OpError:
		switch e.Op {
		case "read", "write", "dial":
			return true
		}
	}
	return false
}

// TODO: this is global, make it configurable
var retries = waitTime{
	duration:      5 * time.Second,
	delayDuration: 1 * time.Second,
}

// waitTime represents sequence waiting for an action to complete successfully.
type waitTime struct {
	duration      time.Duration // total duration of retries.
	delayDuration time.Duration // delay interval between each retry
}

type retryOperation struct {
	waittime waitTime
	last     time.Time
	end      time.Time
	force    bool
	count    int
}

// instantiate new sequence of retries for the given waittime.
func (s waitTime) init() *retryOperation {
	now := time.Now()
	console.Error("Retrying... ")
	return &retryOperation{
		waittime: s,
		last:     now,
		end:      now.Add(s.duration),
		force:    true,
	}
}

func (a *retryOperation) retry() bool {
	now := time.Now()
	sleep := a.nextSleep(now)
	if !a.force && !now.Add(sleep).Before(a.end) {
		console.Infoln() // print a new line
		return false
	}
	a.force = false
	if sleep > 0 && a.count > 0 {
		time.Sleep(sleep)
		now = time.Now()
	}
	a.count++
	a.last = now
	console.Infof("%d ", a.count)
	return true
}

func (a *retryOperation) nextSleep(now time.Time) time.Duration {
	sleep := a.waittime.delayDuration - now.Sub(a.last)
	if sleep < 0 {
		return 0
	}
	return sleep
}
