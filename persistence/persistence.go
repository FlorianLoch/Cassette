package persistence

import (
	"log"

	"github.com/globalsign/mgo"
	// "encoding/gob"
)

const (
	collection = "player-states-per-user"
)

type PlayerStatesDAO struct {
	collection *mgo.Collection
}

func NewPlayerStatesDAO(connectionString string) *PlayerStatesDAO {
	var session, err = mgo.Dial(connectionString)

	if err != nil {
		log.Fatal("Could not reach mongo db!\n", err)
	}

	log.Println("Connected to mongo db!")

	var db = session.DB("")

	var collection = db.C(collection)

	return &PlayerStatesDAO{collection: collection}
}

func (p *PlayerStatesDAO) LoadPlayerStates(userID string) *PlayerStates {
	var item persistenceItem
	var err = p.collection.FindId(userID).One(&item)

	if err != nil {
		if err == mgo.ErrNotFound {
			return &PlayerStates{UserID: userID, States: make([]*PlayerState, 1)}
		}

		log.Fatal("Could not load previous player states from db!\n\t", err)
	}

	return &PlayerStates{UserID: userID, States: item.PlayerStates}
}

func (p *PlayerStatesDAO) SavePlayerStates(playerStates *PlayerStates) {
	var userID = playerStates.UserID

	var wrapped = persistenceItem{Version: "1", UserID: userID, PlayerStates: playerStates.States}

	p.collection.UpsertId(userID, &wrapped)
}

type PlayerState struct {
	PlaybackContextURI string `json:"-"`
	PlaybackItemURI    string `json:"-"`
	TrackName          string `json:"trackName"`
	AlbumName          string `json:"albumName"`
	AlbumArtURL        string `json:"albumArtURL"`
	ArtistName         string `json:"artistName"`
	Progress           int    `json:"progress"`
	Duration           int    `json:"duration"`
}

type persistenceItem struct {
	Version      string
	UserID       string `bson:"_id"`
	PlayerStates []*PlayerState
}

type PlayerStates struct {
	UserID string         `json:"-"`
	States []*PlayerState `json:"states"`
}
