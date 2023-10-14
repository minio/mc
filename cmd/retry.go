package cmd

import (
	"context"
	"fmt"
	"time"

	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
)

type retryManager struct {
	retries       int
	maxRetries    int
	retryInterval time.Duration
	shouldStop    bool
	ctx           context.Context
}

func newRetryManager(ctx context.Context, retryInterval time.Duration, maxRetries int) *retryManager {
	return &retryManager{
		retryInterval: retryInterval,
		maxRetries:    maxRetries,
		ctx:           ctx,
	}
}

type retryMessage struct {
	SourceURL string `json:"sourceURL"`
	TargetURL string `json:"targetURL"`
	Retries   int    `json:"retries"`
}

func (r *retryManager) Stop() {
	r.shouldStop = true
}

func (r retryMessage) String() string {
	return fmt.Sprintf("<INFO> Retries %d: source `%s` >> target `%s`", r.Retries, r.SourceURL, r.TargetURL)
}

func (r retryMessage) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (r *retryManager) encapsulateWithRetry(action func(*retryManager) *probe.Error) {
	for r.retries <= r.maxRetries {
		if r.shouldStop {
			return
		}

		err := action(r)
		if err == nil || r.ctx.Err() != nil {
			return
		}

		<-time.After(r.retryInterval/2 + time.Duration(rand.Int63n(int64(r.retryInterval))))

		r.retries++
	}
}
