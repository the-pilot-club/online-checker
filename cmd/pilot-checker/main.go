package main

import (
	"log"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/the-pilot-club/online-checker/internal/functions"
	"github.com/the-pilot-club/online-checker/internal/store"
	"github.com/the-pilot-club/tpcgo"
)

func main() {

	s, err := tpcgo.NewSession(tpcgo.SessionConfig{
		FCPEnv: "production", // Leaving Blank due to it not being needed
	})
	if err != nil {
		log.Fatalf("failed to create tpcgo session: %v", err)
	}

	st, err := store.NewRedis[functions.PilotSession]("online:", 24*time.Hour)
	if err != nil {
		log.Fatalf("failed to create session store: %v", err)
	}
	defer st.Close()

	for {
		log.Println("Starting Online Checker Process")
		functions.OnlineCheck(s, st)
		log.Println("Online Checker Process Complete. Awaiting Datafeed Update.")
		time.Sleep(15 * time.Second)
	}
}
