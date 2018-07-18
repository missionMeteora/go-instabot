package main

// TODO:
// // FIGRE OUT PAGINATION (MAX ID)
// WE KEEP ITERATING TILL WE HIT CAPS

// FOLLOWERS FEED
// INCORPORATE PROXY (Have a pool which gets assigned on each CreateUser call)
// STATS
// ONE TYPE PER ACCOUNT?
// FOLLOWERS LOGIC
// BILLING

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"github.com/missionMeteora/instabot/misc"
	"github.com/tducasse/goinsta"
	"github.com/tducasse/goinsta/store"

	"github.com/missionMeteora/apiserv"
)

type Server struct {
	r  *apiserv.Server
	db *bolt.DB
}

const (
	BUCKET = "users"
	PORT   = "8080"
)

// Run starts the server
func (srv *Server) Run() error {
	var (
		errCh = make(chan error, 1)
	)

	go func() {
		log.Printf("listening on http://:" + PORT)
		errCh <- srv.r.Run(":" + PORT)
	}()
	return <-errCh
}

func main() {
	// Create path for DB
	os.MkdirAll("./data", 0700)

	rand.Seed(time.Now().UnixNano())
	r := apiserv.New()

	// Open Bolt DB
	db := misc.OpenDB("./data/", "instabot")
	log.Println("Initializing bucket", BUCKET)
	if err := db.Update(func(tx *bolt.Tx) error {
		log.Println("Initializing bucket", BUCKET)
		if _, err := tx.CreateBucketIfNotExists([]byte(BUCKET)); err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		idxStart := uint64(1)
		if err := misc.InitIndex(tx, BUCKET, idxStart); err != nil {
			return err
		}

		return nil
	}); err != nil {
		log.Fatalln(err)
	}

	srv := &Server{
		r:  r,
		db: db,
	}

	blah := Range{Min: 0, Max: 4000}
	lm := Limit{
		MaxRetry: 2,
		Like:     blah,
		Comment:  blah,
		Follow:   blah,
	}

	locations := make(map[string]Cap)
	locations["zoo"] = Cap{
		Like:    5,
		Comment: 5,
		Follow:  5,
	}

	usr := &User{
		Email: "shahzilabid@gmail.com",
		Targets: []*Target{&Target{
			Tags:     locations,
			Limit:    lm,
			Name:     "SHIT",
			Comments: []string{"Where is this?", "I love the zoo"}}},
		Username: "swayshah",
		Password: "muchodinero",
	}
	log.Println("Creating user", SaveUser(usr, db))

	go srv.Act()

	// r.GET("/api", byRange(srv))
	r.GET("/ping", ping(srv))
	// r.GET("/hack", hack(srv))
	// r.GET("/byDay", byDay(srv))

	// Listen and Serve
	if err := srv.Run(); err != nil {
		log.Panicf("Failed to listen: %v", err)
	}

	// // Gets the command line options
	// parseOptions()
	// // Gets the config
	// getConfig()
	// // Tries to login
	// login()
	// if *unfollow {
	// 	syncFollowers()
	// } else if *run {
	// 	// Loop through tags ; follows, likes, and comments, according to the config file
	// 	loopTags()
	// }

}

func ping(s *Server) apiserv.Handler {
	return func(c *apiserv.Context) apiserv.Response {
		return apiserv.PlainResponse("", "pong")
	}
}

func (s *Server) Act() {
	for {
		for _, usr := range GetAllUsers(s.db) {
			// if usr.LastRun > 0 && withinLast(usr.LastRun, 60*60*24) {
			// 	continue
			// }

			// Get config
			var (
				insta *goinsta.Instagram
				err   error
			)
			if len(usr.Session) == 0 || len(usr.Key) == 0 {
				log.Println("NEW!")
				insta = goinsta.NewViaProxy(usr.Username, usr.Password, usr.Proxy)
				if err := insta.Login(); err != nil {
					log.Println("Error logging in", usr.Email, err)
					continue
				}

				key := createKey()
				bytes, err := store.Export(insta, key)
				if err != nil {
					log.Println("Error exporting login", usr.Email, err)
					continue
				}

				usr.Key = key
				usr.Session = bytes
			} else {
				log.Println("OLD!")
				insta, err = usr.GetLogin()
				if err != nil {
					log.Println("Error logging in from import", usr.Email, err)
					continue
				}
			}

			// log.Println("DONE!", len(usr.Session), len(usr.Key))
			for _, tg := range usr.Targets {
				if len(tg.Tags) > 0 {
					loopTags(usr.Username, insta, tg)
				}

				if len(tg.Locations) > 0 {
					loopLocations(usr.Username, insta, tg)
				}

				if len(tg.Followers) > 0 {
				}
			}

			usr.LastRun = time.Now().Unix()

			if err := SaveUser(usr, s.db); err != nil {
				log.Println("Error saving user", usr.Email, err)
				continue
			}

		}

		time.Sleep(20 * time.Minute)
	}
}

func withinLast(timestamp, seconds int64) bool {
	return timestamp >= (time.Now().Unix() - seconds)
}
