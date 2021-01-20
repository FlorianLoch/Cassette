package persistence

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
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

var (
	ErrUserNotFound = errors.New("user not found in db")
)

type PlayerStatesDAO struct {
	collection *mongo.Collection
}

func NewPlayerStatesDAO(collection *mongo.Collection) *PlayerStatesDAO {
	return &PlayerStatesDAO{collection}
}

func NewPlayerStatesDAOFromConnectionString(connectionString string) *PlayerStatesDAO {
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

	dbName := strings.Trim(u.Path, "/")
	if dbName == "" {
		log.Fatal("DB name retrieved from connection string is empty.")
	}

	log.Println(fmt.Sprintf("Connected to mongo db backend! Will use '%s' as db.", dbName))

	collection := client.Database(dbName).Collection(collectionName)

	return NewPlayerStatesDAO(collection)
}

func (p *PlayerStatesDAO) LoadPlayerStates(userID string) *PlayerStates {
	hashedUserID := hashUserID(userID)

	var item persistenceItem
	err := p.collection.FindOne(context.TODO(), bson.D{{Key: "_id", Value: hashedUserID}}).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &PlayerStates{UserID: userID, States: make([]*PlayerState, 0, 1)}
		}

		log.Fatal("Could not load previous player states from db!\n\t", err)
	}

	return &PlayerStates{UserID: userID, States: item.PlayerStates}
}

func (p *PlayerStatesDAO) SavePlayerStates(playerStates *PlayerStates) {
	hashedUserID := hashUserID(playerStates.UserID)

	opts := options.Update().SetUpsert(true)

	_, err := p.collection.UpdateOne(context.TODO(), bson.D{{Key: "_id", Value: hashedUserID}}, bson.D{{Key: "$set", Value: bson.D{{Key: "playerStates", Value: &playerStates.States}, {Key: "version", Value: "2"}}}}, opts)

	if err != nil {
		log.Fatal("Could not write player states to db!\n\t", err)
	}
}

func (p *PlayerStatesDAO) FetchJSONDump(userID string) ([]byte, error) {
	hashedUserID := hashUserID(userID)

	var item persistenceItem
	err := p.collection.FindOne(context.TODO(), bson.D{{Key: "_id", Value: hashedUserID}}).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("Could not load previous player states from db: %w", err)
	}

	json, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("Could not convert record to JSON: %w", err)
	}

	return json, nil
}

func (p *PlayerStatesDAO) DeleteUserRecord(userID string) error {
	hashedUserID := hashUserID(userID)

	res, err := p.collection.DeleteOne(context.TODO(), bson.D{{Key: "_id", Value: hashedUserID}})
	if err != nil {
		return fmt.Errorf("Could not delete user record: %w", err)
	}

	if res.DeletedCount == 0 {
		return ErrUserNotFound
	}

	return nil
}

func hashUserID(userID string) string {
	hash := sha256.Sum256([]byte(userID))
	return fmt.Sprintf("%X", hash)
}

type PlayerState struct {
	PlaybackContextURI string `json:"-" bson:"playbackContextURI"`
	PlaybackItemURI    string `json:"-" bson:"playbackItemURI"`
	TrackName          string `json:"trackName" bson:"trackName"`
	AlbumName          string `json:"albumName" bson:"albumName"`
	AlbumArtURL        string `json:"albumArtURL" bson:"albumArtURL"`
	ArtistName         string `json:"artistName" bson:"artistName"`
	Progress           int    `json:"progress" bson:"progress"`
	Duration           int    `json:"duration" bson:"duration"`
	ShuffleActivated   bool   `json:"shuffleActivated" bson:"shuffleActivated"`
}

type persistenceItem struct {
	Version      string         `bson:"version" json:"version"`
	UserID       string         `bson:"_id" json:"_id"`
	PlayerStates []*PlayerState `bson:"playerStates" json:"playerStates"`
}

type PlayerStates struct {
	UserID string         `json:"-"`
	States []*PlayerState `json:"states" bson:"states"`
}
