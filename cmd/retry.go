package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
)

type retryManager struct {
	retries       int
	maxRetries    int
	retryInterval time.Duration
	commandCtx    context.Context
	retryCtx      context.Context
	cancelRetry   context.CancelFunc
}

func newRetryManager(ctx context.Context, retryInterval time.Duration, maxRetries int) *retryManager {
	retryCtx, cancelFunc := context.WithCancel(context.Background())
	return &retryManager{
		retryInterval: retryInterval,
		maxRetries:    maxRetries,
		commandCtx:    ctx,
		retryCtx:      retryCtx,
		cancelRetry:   cancelFunc,
	}
}

type retryMessage struct {
	SourceURL string `json:"sourceURL"`
	TargetURL string `json:"targetURL"`
	Retries   int    `json:"retries"`
}

func (r retryMessage) String() string {
	return fmt.Sprintf("<INFO> Retries %d: source `%s` >> target `%s`", r.Retries, r.SourceURL, r.TargetURL)
}

func (r retryMessage) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (r *retryManager) retry(action func(rm *retryManager) *probe.Error) {
	defer r.cancelRetry()
	for r.retries <= r.maxRetries {

		err := action(r)
		if err == nil {
			return
		}

		select {
		case <-r.retryCtx.Done():
			return
		case <-r.commandCtx.Done():
			return
		case <-time.After(r.retryInterval/2 + time.Duration(rand.Int63n(int64(r.retryInterval)))):
			r.retries++
		}

	}
}
