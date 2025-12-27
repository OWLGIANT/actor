package repository

import (
	"context"
	"time"

	"github.com/example/microshop/pkg/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepository struct {
	client   *mongo.Client
	database *mongo.Database
	config   *config.MongoDBConfig
}

func NewMongoRepository(cfg *config.MongoDBConfig) (*MongoRepository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, err
	}

	return &MongoRepository{
		client:   client,
		database: client.Database(cfg.Database),
		config:   cfg,
	}, nil
}

func (m *MongoRepository) Ping(ctx context.Context) error {
	return m.client.Ping(ctx, nil)
}

func (m *MongoRepository) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string    `bson:"_id,omitempty"`
	Service   string    `bson:"service"`
	Action    string    `bson:"action"`
	EntityID  string    `bson:"entity_id"`
	Data      bson.M    `bson:"data"`
	CreatedAt time.Time `bson:"created_at"`
}

func (m *MongoRepository) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	collection := m.database.Collection(m.config.Collection)
	log.CreatedAt = time.Now()
	_, err := collection.InsertOne(ctx, log)
	return err
}

func (m *MongoRepository) GetAuditLogs(ctx context.Context, entityID string, limit int64) ([]*AuditLog, error) {
	collection := m.database.Collection(m.config.Collection)

	filter := bson.M{"entity_id": entityID}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit)

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []*AuditLog
	if err = cursor.All(ctx, &logs); err != nil {
		return nil, err
	}

	return logs, nil
}
