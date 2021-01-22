package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/florianloch/spotistate/persistence"

	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

/*
	TODO:
	1. Read in old format
	2. Convert into structs of new format - actually this happens implicitly
	3. Save via code in persistence package, this saves data in new format and also hashed the usernames
	4. Delete the old entries, these are all the ones with "version" field not beeing set. For safety purposes this is done manually.
*/

const (
	collectionName = "player-states-per-user"
)

type PlayerStatesDAO struct {
	collection *mongo.Collection
}

func main() {
	mongoURI := strings.TrimSpace(os.Getenv("mongo_db_uri"))

	if mongoURI == "" {
		log.Fatal("No mongo connection defined. Please set the 'mongo_db_uri' enviroment variable!")
	}

	coll := connect(mongoURI)
	playerStatesDAO := persistence.NewPlayerStatesDAO(coll)

	log.Println("Successfully connected to mongoDB.")

	cursor, err := coll.Find(context.TODO(), bson.D{})

	var results []persistenceItemV1
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
	}

	log.Printf("%d records found.", len(results))

	for _, result := range results {
		statesV2 := make([]*persistence.PlayerState, 0, len(result.PlayerStates))

		for _, stateV1 := range result.PlayerStates {
			stateV2 := &persistence.PlayerState{
				stateV1.PlaybackContextURI,
				stateV1.PlaybackItemURI,
				stateV1.TrackName,
				stateV1.AlbumName,
				stateV1.AlbumArtURL,
				stateV1.ArtistName,
				stateV1.Progress,
				stateV1.Duration,
				stateV1.ShuffleActivated,
			}

			statesV2 = append(statesV2, stateV2)
		}

		statesContainerV2 := persistence.PlayerStates{
			result.UserID,
			statesV2,
		}

		playerStatesDAO.SavePlayerStates(&statesContainerV2)
	}

	log.Println("Done.")
}

func connect(connectionString string) *mongo.Collection {
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

	return client.Database(dbName).Collection(collectionName)
}

type PlayerStateV1 struct {
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

type persistenceItemV1 struct {
	Version      string
	UserID       string           `bson:"_id"`
	PlayerStates []*PlayerStateV1 `bson:"playerstates"`
}

type PlayerStatesV1 struct {
	UserID string           `json:"-"`
	States []*PlayerStateV1 `json:"states"`
}
