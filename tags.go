package main

import (
	"log"
	"time"

	"github.com/tducasse/goinsta"
	"github.com/tducasse/goinsta/response"
)

// Insta is a goinsta.Instagram instance
// var insta *goinsta.Instagram

// login will try to reload a previous session, and will create a new one if it can't
// func login() {
// 	err := reloadSession()
// 	if err != nil {
// 		createAndSaveSession()
// 	}
// }

// func getInput(text string) string {
// 	reader := bufio.NewReader(os.Stdin)
// 	fmt.Printf(text)

// 	input, err := reader.ReadString('\n')
// 	check(err)
// 	return strings.TrimSpace(input)
// }

// Checks if the user is in the slice

// // Logins and saves the session
// func createAndSaveSession() {
// 	insta = goinsta.New(viper.GetString("user.instagram.username"), viper.GetString("user.instagram.password"))
// 	err := insta.Login()
// 	check(err)

// 	key := createKey()
// 	bytes, err := store.Export(insta, key)
// 	check(err)
// 	err = ioutil.WriteFile("session", bytes, 0644)
// 	check(err)
// 	log.Println("Created and saved the session")
// }

// // reloadSession will attempt to recover a previous session
// func reloadSession() error {
// 	if _, err := os.Stat("session"); os.IsNotExist(err) {
// 		return errors.New("No session found")
// 	}

// 	session, err := ioutil.ReadFile("session")
// 	check(err)
// 	log.Println("A session file exists")

// 	key, err := ioutil.ReadFile("key")
// 	check(err)

// 	insta, err = store.Import(session, key)
// 	if err != nil {
// 		return errors.New("Couldn't recover the session")
// 	}

// 	log.Println("Successfully logged in")
// 	return nil

// }

// Go through all the tags in the list
func loopTags(username string, insta *goinsta.Instagram, tg *Target) {
	for tag, value := range tg.Tags {
		// Some converting
		// limits := tg.Restrictions.Tags
		// limits = map[string]int{
		// 	"follow":  int(limitsConf["follow"].(float64)),
		// 	"like":    int(limitsConf["like"].(float64)),
		// 	"comment": int(limitsConf["comment"].(float64)),
		// }
		// What we did so far

		browseTags(username, tag, insta, value, tg.Limit, tg.Comments)
	}
}

// Browses the page for a certain tag, until we reach the limits
func browseTags(username, tag string, insta *goinsta.Instagram, caps Cap, limits Limit, comments []string) {
	var (
		browseIdx    = 0
		numFollowed  = 0
		numLiked     = 0
		numCommented = 0
	)

	for numFollowed < caps.Follow || numLiked < caps.Like || numCommented < caps.Comment {
		log.Println("Fetching the list of images for #" + tag + "\n")
		browseIdx++

		// Getting all the pictures we can on the first page
		// Instagram will return a 500 sometimes, so we will retry 10 times.
		// Check retry() for more info.
		var images response.TagFeedsResponse
		err := retry(10, 20*time.Second, func() (err error) {
			images, err = insta.TagFeed(tag)
			return
		})
		if err != nil {
			log.Println("Err getting tag feed", err)
			continue
		}

		log.Println("Newest", images.Items[0].TakenAt)
		var i = 1
		for _, image := range images.FeedsResponse.Items[0:3] {
			// Exiting the loop if there is nothing left to do
			if numFollowed >= caps.Follow && numLiked >= caps.Like && numCommented >= caps.Comment {
				break
			}

			// Skip our own images
			if image.User.Username == username {
				continue
			}

			// Check if we should fetch new images for tag
			if i >= caps.Follow && i >= caps.Like && i >= caps.Comment {
				break
			}

			// Getting the user info
			// Instagram will return a 500 sometimes, so we will retry 10 times.
			// Check retry() for more info.
			var posterInfo response.GetUsernameResponse
			err := retry(10, 20*time.Second, func() (err error) {
				posterInfo, err = insta.GetUserByID(image.User.ID)
				return
			})
			if err != nil {
				log.Println("Err getting tag feed", err)
				continue
			}

			poster := posterInfo.User
			followerCount := poster.FollowerCount

			// buildLine()

			log.Println("Checking followers for " + poster.Username + " - for #" + tag)
			log.Printf("%s has %d followers\n", poster.Username, followerCount)
			i++

			// Will only follow and comment if we like the picture
			like := followerCount > limits.Like.Min && followerCount < limits.Like.Max && numLiked < caps.Like
			follow := followerCount > limits.Follow.Min && followerCount < limits.Follow.Max && numFollowed < caps.Follow && like
			comment := followerCount > limits.Comment.Min && followerCount < limits.Comment.Max && numCommented < caps.Comment && like

			// Like, then comment/follow
			if like {
				liked := likeImage(insta, image)
				if liked {
					numLiked++
				}

				if follow {
					if followUser(insta, posterInfo) {
						numFollowed++
					}
				}
				if liked && comment {
					commentImage(insta, image, comments)
					numCommented++
				}
			}
			log.Printf("%s done\n\n", poster.Username)

			// This is to avoid the temporary ban by Instagram
			time.Sleep(20 * time.Second)
		}

		if limits.MaxRetry > 0 && browseIdx > limits.MaxRetry {
			log.Println("Currently not enough images for this tag to achieve goals")
			break
		}
	}
}
