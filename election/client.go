// Package election provides a wrapper around a subset of the Election
// functionality of etcd's concurrency package with error handling for common
// failure scenarios such as lease expiration.
package election

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"golang.org/x/net/context"
)

var (
	// ErrCampaignInProgress is returned when a client tries to start a second
	// camapaign if they are either (1) already the leader or (2) not the leader
	// but already campaigning.
	ErrCampaignInProgress = errors.New("election: campaign already in progress")

	// ErrSessionExpired is returned by Campaign() if the underlying session
	// (lease) has expired.
	ErrSessionExpired = errors.New("election: session expired")

	// ErrClientClosed is returned when an election client has been closed and
	// cannot be reused.
	ErrClientClosed = errors.New("election: client has been closed")
)

// Client encapsulates a client of etcd-backed leader elections.
type Client struct {
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

// NewClient returns an election client based on the given etcd client and
// participating in elections rooted at the given prefix. Optional parameters
// can be configured via options, such as configuration of the etcd session TTL.
func NewClient(cli *clientv3.Client, prefix string, options ...ClientOption) (*Client, error) {
	var opts clientOpts
	for _, opt := range options {
		opt(&opts)
	}

	cl := &Client{
		prefix:     prefix,
		opts:       opts,
		etcdClient: cli,
	}

	if err := cl.resetSession(); err != nil {
		return nil, err
	}

	return cl, nil
}

// Campaign starts a new campaign for val at the prefix configured at client
// creation. It blocks until the etcd Campaign call returns, and returns any
// error encountered or ErrSessionExpired if election.Campaign returned a nil
// error but was due to the underlying session expiring. If the client is
// successfully elected with a valid session, a channel is returned which will
// be closed if that session expires in the background. Callers of Campaign()
// should keep a reference to this channel and restart their campaigns if it is
// closed.
//
// If a client is either already campaigning or already elected and has not
// called Resign(), ErrCampaignInProgress will be returned (if the session
// expires in the background the client will be allowed to call Campaign() again
// without calling Resign()). If a session expires either during the call to
// Campaign() or after the call succeeds, calling Campaign() again will create a
// new session transparently for the user.
func (c *Client) Campaign(ctx context.Context, val string) (<-chan struct{}, error) {
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

	c.mu.RLock()
	session := c.session
	c.mu.RUnlock()

	// if we lose the session in the background and that session was the
	// client's active one we want to allow the client to campaign again
	go func(session *concurrency.Session) {
		<-session.Done()
		c.mu.Lock()
		c.resetCampaigning()
		c.cancelWithLock()
		if c.session == session {
			c.session = nil
			c.election = nil
		}
		c.mu.Unlock()
	}(session)

	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.ctxCancel = cancel
	election := c.election
	c.mu.Unlock()

	err := election.Campaign(ctx, val)

	c.mu.Lock()
	c.cancelWithLock()
	c.mu.Unlock()

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

	// defer checking the error from Campaign() until we've returned a
	// SessionExpired error if that was the cause (background routine may have
	// cancelled Campaign() context due to dead session)
	if err != nil {
		return nil, err
	}

	return session.Done(), nil
}

// Resign gives up leadership if the caller was elected. Additionally, if the
// caller was actively campaigning (i.e. a concurrent call to Campaign() was
// still blocking) but had not yet been elected, calling Resign() will cancel
// that ongoing campaign.
func (c *Client) Resign(ctx context.Context) error {
	if c.isClosed() {
		return ErrClientClosed
	}

	defer c.resetCampaigning()

	c.mu.Lock()
	// if we're not the leader but actively campaigning, we only want to cancel
	// the active campaign context (otherwise risk a race between
	// election.Campaign() and election.Resign())
	if c.ctxCancel != nil {
		c.ctxCancel()
		c.ctxCancel = nil
		c.mu.Unlock()
		return nil
	}
	election := c.election
	c.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	err := election.Resign(ctx)
	cancel()

	if err != nil {
		return err
	}

	return nil
}

// Leader returns the value proposed by the currently elected leader of the
// election.
func (c *Client) Leader(ctx context.Context) (string, error) {
	if c.isClosed() {
		return "", ErrClientClosed
	}

	c.mu.RLock()
	election := c.election
	c.mu.RUnlock()

	ctx, cancel := context.WithCancel(ctx)
	ld, err := election.Leader(ctx)
	cancel()

	if err != nil {
		return "", err
	}

	return ld, err
}

// Close closes the client's underlying session and prevents any further
// campaigns from being started.
func (c *Client) Close() error {
	if c.setClosed() {
		c.mu.Lock()
		c.cancelWithLock()
		session := c.session
		c.mu.Unlock()

		return session.Close()
	}

	return nil
}

func (c *Client) resetSession() error {
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

func (c *Client) resetCampaigning() {
	atomic.StoreUint32(&c.campaigning, 0)
}

func (c *Client) isClosed() bool {
	return atomic.LoadUint32(&c.closed) == 1
}

func (c *Client) setClosed() bool {
	return atomic.CompareAndSwapUint32(&c.closed, 0, 1)
}

func (c *Client) cancelWithLock() {
	if c.ctxCancel != nil {
		c.ctxCancel()
		c.ctxCancel = nil
	}
}
