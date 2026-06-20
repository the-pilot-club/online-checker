package functions

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/redis/go-redis/v9"
	"github.com/the-pilot-club/tpcgo"

	_ "github.com/joho/godotenv/autoload"
)

type ATCRedisStore struct {
	CID       string `redis:"cid"`
	Callsign  string `redis:"callsign"`
	Frequency string `redis:"frequency"`
	Start     string `redis:"start"`
	MessageId string `redis:"message_id"`
}

func ATCOnlineCheck(s *tpcgo.Session) {

	dbnum, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		log.Printf("invalid REDIS_DB: %v", err)
		return
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL") + ":6379",
		DB:       dbnum,
		Protocol: 3,
	})
	defer rdb.Close()
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

		if c, found := dfmap[strconv.Itoa(uu.VATSIMCid)]; found {
			opp, _ := rdb.HGet(ctx, "online-atc:"+strconv.Itoa(uu.VATSIMCid), "cid").Result()
			if len(opp) > 0 {
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
			store := []string{
				"cid", strconv.Itoa(c.CID),
				"callsign", c.Callsign,
				"frequency", c.Frequency,
				"start", fmt.Sprintf("<t:%d:f>", time.Now().Unix()),
				"message_id", w.ID,
			}
			if _, reerr := rdb.HSet(ctx, "online-atc:"+strconv.Itoa(uu.VATSIMCid), store).Result(); reerr != nil {
				log.Printf("redis HSet failed for cid %d: %v", uu.VATSIMCid, reerr)
			}
		} else {
			var usr ATCRedisStore
			_ = rdb.HGetAll(ctx, "online-atc:"+strconv.Itoa(uu.VATSIMCid)).Scan(&usr)
			if usr.CID != "" {
				embedd := &discordgo.MessageEmbed{
					Title: "ATC session ended!",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Callsign",
							Value: usr.Callsign,
						},
						{
							Name:  "Frequency",
							Value: usr.Frequency,
						},
						{
							Name:  "Start Time",
							Value: usr.Start,
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
				_, dgerr := d.WebhookMessageEdit(os.Getenv("WEBHOOK_ID"), os.Getenv("WEBHOOK_TOKEN"), usr.MessageId, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embedd},
				})
				if dgerr != nil {
					log.Printf("webhook edit (end) failed for cid %d: %v", uu.VATSIMCid, dgerr)
					continue
				}
				rdb.Del(ctx, "online-atc:"+strconv.Itoa(uu.VATSIMCid))
			}
		}

	}
}
