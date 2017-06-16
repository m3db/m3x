package election

import (
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
)

type clientOpts struct {
	resignTimeout time.Duration
	leaderTimeout time.Duration
	sessionOpts   []concurrency.SessionOption
}

type ClientOption func(*clientOpts)

func WithResignTimeout(to time.Duration) ClientOption {
	return func(o *clientOpts) {
		o.resignTimeout = to
	}
}

func WithLeaderTimeout(to time.Duration) ClientOption {
	return func(o *clientOpts) {
		o.leaderTimeout = to
	}
}

func WithSessionOptions(opts ...concurrency.SessionOption) ClientOption {
	return func(o *clientOpts) {
		o.sessionOpts = opts
	}
}
