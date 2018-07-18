package main

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/boltdb/bolt"
	"github.com/missionMeteora/instabot/misc"
	"github.com/tducasse/goinsta"
	"github.com/tducasse/goinsta/store"
)

type User struct {
	Email   string    `json:"email"`
	Targets []*Target `json:"targets"`

	// Instagram login
	Username string `json:"username"`
	Password string `json:"password"`
	Proxy    string `json:"proxy"`

	Key     []byte `json:"key"`
	Session []byte `json:"session"`

	LastRun int64 `json:"lastRun"`
}

func (usr *User) GetLogin() (*goinsta.Instagram, error) {
	insta, err := store.Import(usr.Session, usr.Key)
	if err != nil {
		log.Println("Error logging in from import", usr.Email, err)
		return nil, err
	}
	insta.Proxy = usr.Proxy
	return insta, nil
}

type Target struct {
	Name string `json:"name"`

	Tags      map[string]Cap `json:"tags"`
	Locations map[string]Cap `json:"locations"`
	Followers map[string]Cap `json:"followers"`

	Limit    Limit    `json:"limits"`
	Comments []string `json:"comments"`
}

type Limit struct {
	MaxRetry int   `json:"maxRetry"`
	Like     Range `json:"like"`
	Comment  Range `json:"comment"`
	Follow   Range `json:"follow"`
}

type Range struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type Cap struct {
	Like    int `json:"like"`
	Comment int `json:"comment"`
	Follow  int `json:"follow"`
}

func (t *Target) Matches(inc *Target) bool {
	if t.Name == inc.Name {
		return true
	}
	return false
}

func SaveUser(usr *User, db *bolt.DB) error {
	if usr == nil || usr.Email == "" {
		return errors.New("No user information")
	}

	// if usr.Proxy == "" {
	// 	return errors.New("No proxy information")
	// }

	if usr.Username == "" || usr.Password == "" {
		return errors.New("No login information")
	}

	if err := db.Update(func(tx *bolt.Tx) (err error) {
		var sv []byte
		if sv, err = json.Marshal(&usr); err != nil {
			return
		}

		return misc.PutBucketBytes(tx, BUCKET, usr.Email, sv)
	}); err != nil {
		return err
	}

	return nil
}

func GetAllUsers(db *bolt.DB) []*User {
	st := []*User{}
	if err := db.View(func(tx *bolt.Tx) (err error) {
		tx.Bucket([]byte(BUCKET)).ForEach(func(cid, b []byte) (err error) {
			var usr User
			if err := json.Unmarshal(b, &usr); err != nil {
				log.Println("error when unmarshalling users", string(b))
				return nil
			}

			st = append(st, &usr)
			return
		})
		return nil
	}); err != nil {
		return nil
	}

	return st
}

func GetUser(id string, db *bolt.DB) (*User, error) {
	var user User
	if err := db.View(func(tx *bolt.Tx) (err error) {
		b := tx.Bucket([]byte(BUCKET)).Get([]byte(id))
		if err = json.Unmarshal(b, &user); err != nil && len(b) > 0 {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &user, nil
}

func GetTargets(id string, db *bolt.DB) ([]*Target, error) {
	var user User
	if err := db.View(func(tx *bolt.Tx) (err error) {
		b := tx.Bucket([]byte(BUCKET)).Get([]byte(id))
		if err = json.Unmarshal(b, &user); err != nil && len(b) > 0 {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return user.Targets, nil
}

func Insert(id string, tg *Target, db *bolt.DB) error {
	if err := db.Update(func(tx *bolt.Tx) (err error) {
		b := tx.Bucket([]byte(BUCKET))

		var usr User
		ov := b.Get([]byte(id))
		if len(ov) == 0 {
			return errors.New("User not found")
		}

		if err = json.Unmarshal(ov, &usr); err != nil && len(ov) > 0 {
			return err
		}

		usr.Targets = append(usr.Targets, tg)

		var sv []byte
		if sv, err = json.Marshal(&usr); err != nil {
			return
		}

		if err = misc.PutBucketBytes(tx, BUCKET, id, sv); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func Delete(id string, tg *Target, db *bolt.DB) error {
	if err := db.Update(func(tx *bolt.Tx) (err error) {
		b := tx.Bucket([]byte(BUCKET))

		var usr User
		ov := b.Get([]byte(id))
		if len(ov) == 0 {
			return errors.New("User not found")
		}

		if err = json.Unmarshal(ov, &usr); err != nil && len(ov) > 0 {
			return err
		}

		var filtered []*Target
		for _, t := range usr.Targets {
			if !t.Matches(tg) {
				filtered = append(filtered, t)
			}
		}

		usr.Targets = filtered

		var sv []byte
		if sv, err = json.Marshal(&usr); err != nil {
			return
		}

		if err = misc.PutBucketBytes(tx, BUCKET, id, sv); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
