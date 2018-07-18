package main

import (
	"log"
	"time"

	"github.com/tducasse/goinsta"
	"github.com/tducasse/goinsta/response"
)

// Go through all the tags in the list
func loopFollowers(username string, insta *goinsta.Instagram, tg *Target) {
	for name, value := range tg.Followers {
		// Some converting
		// limits := tg.Restrictions.Tags
		// limits = map[string]int{
		// 	"follow":  int(limitsConf["follow"].(float64)),
		// 	"like":    int(limitsConf["like"].(float64)),
		// 	"comment": int(limitsConf["comment"].(float64)),
		// }
		// What we did so far

		browseFollowers(username, name, insta, value, tg.Limit, tg.Comments)
	}
}

// Browses the page for a certain tag, until we reach the limits
func browseFollowers(username, targetName string, insta *goinsta.Instagram, caps Cap, limits Limit, comments []string) {
	var (
		browseIdx    = 0
		numFollowed  = 0
		numLiked     = 0
		numCommented = 0
	)

	// Get this user's info
	usr, err := insta.GetUserByUsername(targetName)
	if err != nil {
		log.Println("Cannot find username", username, targetName)
		return
	}

	// Get follower list big enough to satisfy highest value
	max := caps.Follow
	if caps.Like > max {
		max = caps.Like
	}
	if caps.Comment > max {
		max = caps.Comment
	}

	// GET FOLLOWERS THAT SATISFY MAX
	var followers response.UsersResponse
	var (
		last response.UsersResponse
	)
	for i := 0; i < max; i++ {
		log.Println("MAX ID", last.NextMaxID)
		last, err = insta.UserFollowers(usr.User.ID, last.NextMaxID)
		if err != nil {
			log.Println("Cannot get follower feed", err)
			return
		}
		followers.Users = append(followers.Users, last.Users...)
		log.Println("LEN OF FOLLOWERS", len(followers.Users))
		time.Sleep(20 * time.Second)
	}

	for numFollowed < caps.Follow || numLiked < caps.Like || numCommented < caps.Comment {
		log.Println("Fetching the list of images for " + targetName + "\n")
		browseIdx++

		// Getting all the pictures we can on the first page
		// Instagram will return a 500 sometimes, so we will retry 10 times.
		// Check retry() for more info.
		var (
			images []response.UserFeedResponse
			tmp    response.UserFeedResponse
		)
		for _, flwr := range followers.Users {
			err := retry(10, 20*time.Second, func() (err error) {
				tmp, err = insta.LatestUserFeed(flwr.ID)
				if len(tmp.Items) > 0 {
					// Grab first image of every feed
					images = append(images, tmp)
				}
				return
			})
			time.Sleep(21 * time.Second)
			if err != nil {
				log.Println("Err getting tag feed", err)
				continue
			}
		}

		var i = 1
		for _, resp := range images {
			for _, image := range resp.Items {
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

				log.Println("Checking followers for " + poster.Username + " - for " + locName)
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
		}

		if limits.MaxRetry > 0 && browseIdx > limits.MaxRetry {
			log.Println("Currently not enough images for this tag to achieve goals")
			break
		}
	}
}
