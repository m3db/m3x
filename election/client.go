// Package election provides a wrapper around a subset of the Election
// functionality of etcd's concurrency package with error handling for common
// failure scenarios.
package election

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"golang.org/x/net/context"
)

var (
	ErrCampaignInProgress = errors.New("election: campaign already in progress")
	ErrSessionExpired     = errors.New("election: session expired")
	ErrClientClosed       = errors.New("election: client has been closed")
)

const (
	DefaultLeaderTimeout = 30 * time.Second
	DefaultResignTimeout = 30 * time.Second
)

type client struct {
	mu sync.RWMutex

	prefix string
	opts   clientOpts

	etcdClient *clientv3.Client
	election   *concurrency.Election
	session    *concurrency.Session

	campaigning uint32
	closed      uint32
	ctxCancel   context.CancelFunc
}

func NewClient(cli *clientv3.Client, prefix string, options ...ClientOption) (*client, error) {
	opts := clientOpts{
		leaderTimeout: DefaultLeaderTimeout,
		resignTimeout: DefaultResignTimeout,
	}

	for _, opt := range options {
		opt(&opts)
	}

	cl := &client{
		prefix:     prefix,
		opts:       opts,
		etcdClient: cli,
	}

	if err := cl.resetSession(); err != nil {
		return nil, err
	}

	return cl, nil
}

func (c *client) Campaign(ctx context.Context, val string) (<-chan struct{}, error) {
	if c.isClosed() {
		return nil, ErrClientClosed
	}

	if !atomic.CompareAndSwapUint32(&c.campaigning, 0, 1) {
		return nil, ErrCampaignInProgress
	}

	c.mu.RLock()
	reset := c.session == nil || c.election == nil
	c.mu.RUnlock()
	if reset {
		if err := c.resetSession(); err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.ctxCancel = cancel
	c.mu.Unlock()

	err := c.election.Campaign(ctx, val)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	session := c.session
	c.mu.RUnlock()

	select {
	case <-session.Done():
		// may get nil error from election.Campaign() but was not elected due to
		// session expiration
		c.mu.Lock()
		c.session = nil
		c.election = nil
		c.mu.Unlock()
		c.resetCampaigning()

		return nil, ErrSessionExpired
	default:
	}

	c.mu.Lock()
	c.cancelWithLock()
	c.mu.Unlock()

	// if we lose the session in the background and that session was the
	// client's active one we want to allow the client to campaign again
	go func(session *concurrency.Session) {
		<-session.Done()
		c.mu.Lock()
		c.resetCampaigning()
		if c.session == session {
			c.session = nil
			c.election = nil
		}
		c.mu.Unlock()
	}(session)

	return session.Done(), nil
}

func (c *client) Resign(ctx context.Context) error {
	if c.isClosed() {
		return ErrClientClosed
	}

	c.mu.Lock()
	// if we're not the leader but actively campaigning, we also want to cancel
	// the active campaign
	c.cancelWithLock()
	election := c.election
	c.mu.Unlock()

	defer c.resetCampaigning()

	ctx, cancel := context.WithTimeout(ctx, c.opts.resignTimeout)
	err := election.Resign(ctx)
	cancel()

	if err != nil {
		return err
	}

	return nil
}

func (c *client) Leader(ctx context.Context) (string, error) {
	if c.isClosed() {
		return "", ErrClientClosed
	}

	c.mu.RLock()
	election := c.election
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, c.opts.leaderTimeout)
	ld, err := election.Leader(ctx)
	cancel()

	if err != nil {
		return "", err
	}

	return ld, err
}

func (c *client) Close() error {
	if c.setClosed() {
		c.mu.Lock()
		c.cancelWithLock()
		session := c.session
		c.mu.Unlock()

		return session.Close()
	}

	return nil
}

func (c *client) resetSession() error {
	session, err := concurrency.NewSession(c.etcdClient, c.opts.sessionOpts...)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.session = session
	c.election = concurrency.NewElection(session, c.prefix)
	c.mu.Unlock()
	return nil
}

func (c *client) isCampaigning() bool {
	return atomic.LoadUint32(&c.campaigning) == 1
}

func (c *client) resetCampaigning() {
	atomic.StoreUint32(&c.campaigning, 0)
}

func (c *client) isClosed() bool {
	return atomic.LoadUint32(&c.closed) == 1
}

func (c *client) setClosed() bool {
	return atomic.CompareAndSwapUint32(&c.closed, 0, 1)
}

func (c *client) cancelWithLock() {
	if c.ctxCancel != nil {
		c.ctxCancel()
		c.ctxCancel = nil
	}
}
