package persistence

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	collectionName = "player-states-per-user"
)

type PlayerStatesDAO struct {
	collection *mongo.Collection
}

func NewPlayerStatesDAO(connectionString string) *PlayerStatesDAO {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(connectionString))

	if err == nil {
		err = client.Ping(context.Background(), readpref.Primary())
	}

	if err != nil {
		log.Fatal("Could not reach mongo db!\nTried to connect at: ", connectionString, "\nBut got error: ", err)
	}

	u, err := url.Parse(connectionString)
	if err != nil {
		log.Fatal("Could not fetch db name from connection string: ", err)
	}

	var dbName = strings.Trim(u.Path, "/")
	if dbName == "" {
		log.Fatal("DB name retrieved from connection string is empty.")
	}

	log.Println(fmt.Sprintf("Connected to mongo db backend! Will use '%s' as db.", dbName))

	var collection = client.Database(dbName).Collection(collectionName)

	return &PlayerStatesDAO{collection: collection}
}

func (p *PlayerStatesDAO) LoadPlayerStates(userID string) *PlayerStates {
	var item persistenceItem
	var err = p.collection.FindOne(context.TODO(), bson.D{{"_id", userID}}).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &PlayerStates{UserID: userID, States: make([]*PlayerState, 0, 1)}
		}

		log.Fatal("Could not load previous player states from db!\n\t", err)
	}

	return &PlayerStates{UserID: userID, States: item.PlayerStates}
}

func (p *PlayerStatesDAO) SavePlayerStates(playerStates *PlayerStates) {
	var userID = playerStates.UserID

	opts := options.Update().SetUpsert(true)

	_, err := p.collection.UpdateOne(context.TODO(), bson.D{{"_id", userID}}, bson.D{{"$set", bson.D{{"playerstates", &playerStates.States}}}}, opts)

	if err != nil {
		log.Fatal("Could not write player states to db!\n\t", err)
	}
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
	ShuffleActivated   bool   `json:"shuffleActivated"`
}

type persistenceItem struct {
	Version      string
	UserID       string         `bson:"_id"`
	PlayerStates []*PlayerState `bson:"playerstates"`
}

type PlayerStates struct {
	UserID string         `json:"-"`
	States []*PlayerState `json:"states"`
}
