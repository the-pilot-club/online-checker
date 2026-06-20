package functions

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/the-pilot-club/online-checker/internal/store"
	"github.com/the-pilot-club/tpcgo"

	_ "github.com/joho/godotenv/autoload"
)

// ATCSession is the persisted state for an online ATC session. The redis tags
// are used by the Redis store backend and ignored by others.
type ATCSession struct {
	CID       string `redis:"cid"`
	Callsign  string `redis:"callsign"`
	Frequency string `redis:"frequency"`
	Start     string `redis:"start"`
	MessageId string `redis:"message_id"`
}

// atcHandler implements Handler for VATSIM controllers.
type atcHandler struct{}

func (atcHandler) Username() string { return "TPC ATC Tracking" }

// ShouldAnnounce announces every controller that comes online.
func (atcHandler) ShouldAnnounce(tpcgo.Controller) bool { return true }

func (atcHandler) Online(feed *tpcgo.DataFeed) map[string]tpcgo.Controller {
	online := make(map[string]tpcgo.Controller, len(feed.Controllers))
	for _, c := range feed.Controllers {
		online[strconv.Itoa(c.CID)] = c
	}
	return online
}

func (atcHandler) StartEmbed(c tpcgo.Controller, now time.Time) (*discordgo.MessageEmbed, ATCSession) {
	embed := newEmbed("ATC has gone online!",
		field("Callsign", c.Callsign),
		field("Frequency", c.Frequency),
		field("Start Time", discordTime(now)),
	)
	session := ATCSession{
		CID:       strconv.Itoa(c.CID),
		Callsign:  c.Callsign,
		Frequency: c.Frequency,
		Start:     discordTime(now),
	}
	return embed, session
}

func (atcHandler) EndEmbed(s ATCSession, now time.Time) *discordgo.MessageEmbed {
	return newEmbed("ATC session ended!",
		field("Callsign", s.Callsign),
		field("Frequency", s.Frequency),
		field("Start Time", s.Start),
		field("End Time", discordTime(now)),
	)
}

// UpdateEmbed is a no-op: controller sessions carry no updatable state.
func (atcHandler) UpdateEmbed(tpcgo.Controller, ATCSession) (*discordgo.MessageEmbed, ATCSession, bool) {
	return nil, ATCSession{}, false
}

func (atcHandler) MessageID(s ATCSession) string { return s.MessageId }

func (atcHandler) SetMessageID(s ATCSession, id string) ATCSession {
	s.MessageId = id
	return s
}

// ATCOnlineCheck performs one reconciliation pass for online controllers.
func ATCOnlineCheck(src DataSource, st store.SessionStore[ATCSession]) {
	webhook, err := newDiscordWebhook()
	if err != nil {
		log.Printf("failed to create discord webhook: %v", err)
		return
	}
	NewTracker[tpcgo.Controller, ATCSession](st, webhook, atcHandler{}).Run(context.Background(), src)
}
