package functions

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/the-pilot-club/online-checker/internal/store"
	"github.com/the-pilot-club/tpcgo"
)

// DataSource provides the inputs a tracker needs each pass: the CIDs to watch
// and the live VATSIM data feed. It is satisfied by *tpcgo.Session.
type DataSource interface {
	GetAllFCPUsersCID() ([]*tpcgo.FCPCIDOnly, error)
	GetVatsimDataFeed() (*tpcgo.DataFeed, error)
}

// Handler captures the entity-specific behaviour (ATC vs pilot) that the
// generic tracker delegates to. E is the live data-feed entity and S is the
// persisted session.
type Handler[E any, S any] interface {
	// Online returns the currently-online entities keyed by CID string.
	Online(feed *tpcgo.DataFeed) map[string]E
	// Username is the webhook display name used for announcements.
	Username() string
	// ShouldAnnounce reports whether a newly online entity warrants an announcement.
	ShouldAnnounce(e E) bool
	// StartEmbed builds the announcement embed and the session to persist for a
	// newly online entity. The returned session need not carry a message ID.
	StartEmbed(e E, now time.Time) (*discordgo.MessageEmbed, S)
	// EndEmbed builds the closing embed for a session whose entity went offline.
	EndEmbed(existing S, now time.Time) *discordgo.MessageEmbed
	// UpdateEmbed builds an updated embed and session for an already-announced
	// entity still online; ok is false when nothing needs to change.
	UpdateEmbed(e E, existing S) (embed *discordgo.MessageEmbed, updated S, ok bool)
	// MessageID returns the Discord message ID stored in a session.
	MessageID(s S) string
	// SetMessageID returns a copy of s with its message ID set.
	SetMessageID(s S, id string) S
}

// Tracker reconciles persisted sessions against the live VATSIM feed,
// announcing, updating and closing out sessions via a Webhook.
type Tracker[E any, S any] struct {
	store   store.SessionStore[S]
	webhook Webhook
	handler Handler[E, S]
	now     func() time.Time
}

// NewTracker wires a tracker to its store, webhook and handler.
func NewTracker[E any, S any](st store.SessionStore[S], webhook Webhook, handler Handler[E, S]) *Tracker[E, S] {
	return &Tracker[E, S]{store: st, webhook: webhook, handler: handler, now: time.Now}
}

// Run performs a single reconciliation pass over all watched CIDs.
func (t *Tracker[E, S]) Run(ctx context.Context, src DataSource) {
	users, err := src.GetAllFCPUsersCID()
	if err != nil {
		log.Println("error getting users:", err)
		return
	}
	feed, err := src.GetVatsimDataFeed()
	if err != nil {
		log.Println("error getting vatsim data:", err)
		return
	}

	online := t.handler.Online(feed)
	now := t.now()
	for _, u := range users {
		t.reconcile(ctx, strconv.Itoa(u.VATSIMCid), online, now)
	}
}

// reconcile resolves a single CID: announce, update, close out or ignore.
func (t *Tracker[E, S]) reconcile(ctx context.Context, cid string, online map[string]E, now time.Time) {
	existing, found, err := t.store.Get(ctx, cid)
	if err != nil {
		log.Printf("store get failed for cid %s: %v", cid, err)
		return
	}

	entity, isOnline := online[cid]
	switch {
	case !isOnline:
		if found {
			t.end(ctx, cid, existing, now)
		}
	case !found:
		t.announce(ctx, cid, entity, now)
	default:
		t.update(ctx, cid, entity, existing)
	}
}

// announce posts a new session and persists it.
func (t *Tracker[E, S]) announce(ctx context.Context, cid string, entity E, now time.Time) {
	if !t.handler.ShouldAnnounce(entity) {
		return
	}
	embed, session := t.handler.StartEmbed(entity, now)
	id, err := t.webhook.Send(t.handler.Username(), embed)
	if err != nil {
		log.Printf("webhook send failed for cid %s: %v", cid, err)
		return
	}
	session = t.handler.SetMessageID(session, id)
	if err := t.store.Set(ctx, cid, session); err != nil {
		log.Printf("store set failed for cid %s: %v", cid, err)
	}
}

// update edits an already-announced session when the handler reports a change.
func (t *Tracker[E, S]) update(ctx context.Context, cid string, entity E, existing S) {
	embed, updated, ok := t.handler.UpdateEmbed(entity, existing)
	if !ok {
		return
	}
	if err := t.webhook.Edit(t.handler.MessageID(existing), embed); err != nil {
		log.Printf("webhook edit failed for cid %s: %v", cid, err)
		return
	}
	if err := t.store.Set(ctx, cid, updated); err != nil {
		log.Printf("store set (update) failed for cid %s: %v", cid, err)
	}
}

// end closes out a session whose entity has gone offline.
func (t *Tracker[E, S]) end(ctx context.Context, cid string, existing S, now time.Time) {
	embed := t.handler.EndEmbed(existing, now)
	if err := t.webhook.Edit(t.handler.MessageID(existing), embed); err != nil {
		log.Printf("webhook edit (end) failed for cid %s: %v", cid, err)
		return
	}
	if err := t.store.Delete(ctx, cid); err != nil {
		log.Printf("store delete failed for cid %s: %v", cid, err)
	}
}
