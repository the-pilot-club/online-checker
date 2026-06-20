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

	st, err := store.NewRedis[functions.ATCSession]("online-atc:", 24*time.Hour)
	if err != nil {
		log.Fatalf("failed to create session store: %v", err)
	}
	defer st.Close()

	for {
		log.Println("Starting ATC Online Checker Process")
		functions.ATCOnlineCheck(s, st)
		log.Println("ATC Online Checker Process Complete. Awaiting Datafeed Update.")
		time.Sleep(15 * time.Second)
	}
}
