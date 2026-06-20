package functions

import (
	"context"
	"fmt"
	"log"
	"os"
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

func ATCOnlineCheck(s *tpcgo.Session, st store.SessionStore[ATCSession]) {

	ctx := context.Background()

	d, err := discordgo.New("")
	if err != nil {
		log.Printf("failed to create discord client: %v", err)
		return
	}

	u, err := s.GetAllFCPUsersCID()
	if err != nil {
		log.Println("Error getting users: ", err)
		return
	}
	o, err := s.GetVatsimDataFeed()
	if err != nil {
		log.Println("Error getting vatsim data", err)
		return
	}

	var dfmap = make(map[string]tpcgo.Controller)

	for _, v := range o.Controllers {
		dfmap[strconv.Itoa(v.CID)] = v
	}

	for _, uu := range u {

		cid := strconv.Itoa(uu.VATSIMCid)

		existing, found, err := st.Get(ctx, cid)
		if err != nil {
			log.Printf("store get failed for cid %d: %v", uu.VATSIMCid, err)
			continue
		}

		c, online := dfmap[cid]
		if !online {
			// The controller is no longer connected: close out the session.
			if found {
				embedd := &discordgo.MessageEmbed{
					Title: "ATC session ended!",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Callsign",
							Value: existing.Callsign,
						},
						{
							Name:  "Frequency",
							Value: existing.Frequency,
						},
						{
							Name:  "Start Time",
							Value: existing.Start,
						},
						{
							Name:  "End Time",
							Value: fmt.Sprintf("<t:%d:f>", time.Now().Unix()),
						},
					},
					Color: 3651327,
					Footer: &discordgo.MessageEmbedFooter{
						Text:    "Made by the TPC Tech Team",
						IconURL: "https://static1.squarespace.com/static/614689d3918044012d2ac1b4/t/616ff36761fabc72642806e3/1634726781251/TPC_FullColor_TransparentBg_1280x1024_72dpi.png",
					}}
				_, dgerr := d.WebhookMessageEdit(os.Getenv("WEBHOOK_ID"), os.Getenv("WEBHOOK_TOKEN"), existing.MessageId, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embedd},
				})
				if dgerr != nil {
					log.Printf("webhook edit (end) failed for cid %d: %v", uu.VATSIMCid, dgerr)
					continue
				}
				if derr := st.Delete(ctx, cid); derr != nil {
					log.Printf("store delete failed for cid %d: %v", uu.VATSIMCid, derr)
				}
			}
			continue
		}

		// Already announced: nothing to do.
		if found {
			continue
		}

		embedstart := &discordgo.MessageEmbed{
			Title: "ATC has gone online!",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Callsign",
					Value: c.Callsign,
				},
				{
					Name:  "Frequency",
					Value: c.Frequency,
				},
				{
					Name:  "Start Time",
					Value: fmt.Sprintf("<t:%d:f>", time.Now().Unix()),
				},
			},
			Color: 3651327,
			Footer: &discordgo.MessageEmbedFooter{
				Text:    "Made by the TPC Tech Team",
				IconURL: "https://static1.squarespace.com/static/614689d3918044012d2ac1b4/t/616ff36761fabc72642806e3/1634726781251/TPC_FullColor_TransparentBg_1280x1024_72dpi.png",
			}}
		w, dgerr := d.WebhookExecute(os.Getenv("WEBHOOK_ID"), os.Getenv("WEBHOOK_TOKEN"), true, &discordgo.WebhookParams{
			Embeds:    []*discordgo.MessageEmbed{embedstart},
			AvatarURL: "https://cdn.thepilotclub.org/fcp/tpc%20logo.png",
			Username:  "TPC ATC Tracking",
		})
		if dgerr != nil {
			log.Printf("webhook execute failed for cid %d: %v", uu.VATSIMCid, dgerr)
			continue
		}

		session := ATCSession{
			CID:       strconv.Itoa(c.CID),
			Callsign:  c.Callsign,
			Frequency: c.Frequency,
			Start:     fmt.Sprintf("<t:%d:f>", time.Now().Unix()),
			MessageId: w.ID,
		}
		if serr := st.Set(ctx, cid, session); serr != nil {
			log.Printf("store set failed for cid %d: %v", uu.VATSIMCid, serr)
		}
	}
}
