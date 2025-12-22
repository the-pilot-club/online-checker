package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/the-pilot-club/tpcgo"

	_ "github.com/joho/godotenv/autoload"
)

func OnlineCheck(s *tpcgo.Session, err error) {

	dbnum, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		panic(err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL") + ":6379",
		DB:       dbnum,
		Protocol: 3,
	})
	ctx := context.Background()

	//d, err := discordgo.New("")

	u, err := s.GetAllFCPUsersCID()
	if err != nil {
		fmt.Println("Error getting users: ", err)
		return
	}
	o, err := s.GetVatsimDataFeed()
	if err != nil {
		fmt.Println("Error getting vatsim data", err)
		return
	}

	var dfmap = make(map[string]interface{})

	for _, v := range o.Pilots {
		dfmap[strconv.Itoa(v.CID)] = v
	}

	for _, uu := range u {
		// CHeck Redis for cid
		op, _ := rdb.HGet(ctx, "online:"+strconv.Itoa(uu.VATSIMCid), "cid").Result()
		if len(op) > 0 {
			if _, found := dfmap[op]; found {
				continue
			} else {
				fmt.Println("deleting new member: " + strconv.Itoa(uu.VATSIMCid))
				rdb.Del(ctx, "online:"+strconv.Itoa(uu.VATSIMCid))
				// TODO: Post Offline Message
			}
		}
		if value, found := dfmap[strconv.Itoa(uu.VATSIMCid)]; found {
			opp, _ := rdb.HGet(ctx, "online:"+strconv.Itoa(uu.VATSIMCid), "cid").Result()
			//fmt.Println(opp)

			if len(opp) > 0 {
				continue
			} else {
				var v = value.(tpcgo.Pilot)
				if v.FlightPlan != nil {
					if strings.Contains(v.FlightPlan.Remarks, "CALSIGN PILOT CLUB") {
						_, reerr := rdb.HSet(ctx, "online:"+strconv.Itoa(uu.VATSIMCid), []string{
							"cid", strconv.Itoa(v.CID),
							"callsign", v.Callsign,
							"start", v.LogonTime,
						}).Result()
						if reerr != nil {
							fmt.Println(err)
						}
						// TODO: Post Online Message
					}
				}
			}
		}

	}
	err = rdb.Close()
	if err != nil {
		return
	}

}
