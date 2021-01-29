package persistence

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"context"

	"github.com/rs/zerolog/log"
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

func Connect(connectionString string) (*PlayerStatesDAO, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(connectionString))
	if err == nil {
		err = client.Ping(context.Background(), readpref.Primary())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB at '%s': %w", connectionString, err)
	}

	u, err := url.Parse(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse given connection string '%s': %w", connectionString, err)
	}

	dbName := strings.Trim(u.Path, "/")
	if dbName == "" {
		return nil, fmt.Errorf("given database name is empty '%s'", connectionString)
	}

	log.Info().Msgf("Connected to mongo db backend! Will use '%s' as db.", dbName)

	collection := client.Database(dbName).Collection(collectionName)

	return &PlayerStatesDAO{collection}, nil
}

func (p *PlayerStatesDAO) LoadPlayerStates(userID string) ([]*PlayerState, error) {
	hashedUserID := hashUserID(userID)

	var item persistenceItem
	err := p.collection.FindOne(context.TODO(), bson.D{{Key: "_id", Value: hashedUserID}}).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return make([]*PlayerState, 0), nil
		}

		return nil, err
	}

	return item.PlayerStates, nil
}

func (p *PlayerStatesDAO) SavePlayerStates(userID string, playerStates []*PlayerState) error {
	hashedUserID := hashUserID(userID)

	opts := options.Update().SetUpsert(true)

	_, err := p.collection.UpdateOne(context.TODO(), bson.D{{Key: "_id", Value: hashedUserID}}, bson.D{{Key: "$set", Value: bson.D{{Key: "playerStates", Value: playerStates}, {Key: "version", Value: "2"}}}}, opts)

	if err != nil {
		return err
	}

	return nil
}

func (p *PlayerStatesDAO) FetchJSONDump(userID string) ([]byte, error) {
	hashedUserID := hashUserID(userID)

	var item persistenceItem
	err := p.collection.FindOne(context.TODO(), bson.D{{Key: "_id", Value: hashedUserID}}).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("could not load previous player states from db: %w", err)
	}

	json, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("could not convert record to JSON: %w", err)
	}

	return json, nil
}

func (p *PlayerStatesDAO) DeleteUserRecord(userID string) error {
	hashedUserID := hashUserID(userID)

	res, err := p.collection.DeleteOne(context.TODO(), bson.D{{Key: "_id", Value: hashedUserID}})
	if err != nil {
		return fmt.Errorf("could not delete user record: %w", err)
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
