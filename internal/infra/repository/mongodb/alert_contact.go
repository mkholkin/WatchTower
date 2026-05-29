package mongodb

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
)

type alertContactRepositoryMG struct {
	coll *mongo.Collection
	log  *slog.Logger
}

func NewAlertContactRepository(db *mongo.Database, logger *slog.Logger) repo.AlertContactRepository {
	return &alertContactRepositoryMG{
		coll: db.Collection(collAlertContacts),
		log:  logger.With("component", "mongodb_alert_contact_repository"),
	}
}

func (r *alertContactRepositoryMG) Create(ctx context.Context, contact *alert.Contact) error {
	cfgJSON, err := json.Marshal(contact.Config)
	if err != nil {
		return err
	}
	_, err = r.coll.InsertOne(ctx, bson.M{
		"_id":        contact.ID.String(),
		"user_login": contact.User.Login,
		"type":       string(contact.Type),
		"name":       contact.Name,
		"config":     string(cfgJSON),
		"is_active":  contact.IsActive,
	})
	return mapMongoError(err)
}

func (r *alertContactRepositoryMG) GetByID(ctx context.Context, id uuid.UUID) (*alert.Contact, error) {
	return r.findOne(ctx, bson.M{"_id": id.String()})
}

func (r *alertContactRepositoryMG) Update(ctx context.Context, contact *alert.Contact) error {
	cfgJSON, err := json.Marshal(contact.Config)
	if err != nil {
		return err
	}
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": contact.ID.String()}, bson.M{"$set": bson.M{
		"user_login": contact.User.Login,
		"type":       string(contact.Type),
		"name":       contact.Name,
		"config":     string(cfgJSON),
		"is_active":  contact.IsActive,
	}})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *alertContactRepositoryMG) DeleteByID(ctx context.Context, id uuid.UUID) error {
	res, err := r.coll.DeleteOne(ctx, bson.M{"_id": id.String()})
	if err != nil {
		return mapMongoError(err)
	}
	if res.DeletedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *alertContactRepositoryMG) GetByIDBulk(ctx context.Context, ids []uuid.UUID) ([]alert.Contact, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = id.String()
	}

	cursor, err := r.coll.Find(ctx, bson.M{"_id": bson.M{"$in": strIDs}})
	if err != nil {
		return nil, mapMongoError(err)
	}
	defer cursor.Close(ctx)

	var docs []alertContactDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, mapMongoError(err)
	}

	result := make([]alert.Contact, len(docs))
	for i, d := range docs {
		c, err := docToAlertContact(d)
		if err != nil {
			return nil, err
		}
		result[i] = *c
	}
	return result, nil
}

func (r *alertContactRepositoryMG) Enable(ctx context.Context, id uuid.UUID) error {
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": id.String()}, bson.M{"$set": bson.M{"is_active": true}})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *alertContactRepositoryMG) Disable(ctx context.Context, id uuid.UUID) error {
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": id.String()}, bson.M{"$set": bson.M{"is_active": false}})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *alertContactRepositoryMG) GetByUserLogin(ctx context.Context, userLogin string) ([]alert.Contact, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"user_login": userLogin})
	if err != nil {
		return nil, mapMongoError(err)
	}
	defer cursor.Close(ctx)

	var docs []alertContactDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, mapMongoError(err)
	}

	result := make([]alert.Contact, len(docs))
	for i, d := range docs {
		c, err := docToAlertContact(d)
		if err != nil {
			return nil, err
		}
		result[i] = *c
	}
	return result, nil
}

func (r *alertContactRepositoryMG) findOne(ctx context.Context, filter bson.M) (*alert.Contact, error) {
	var doc alertContactDoc
	if err := r.coll.FindOne(ctx, filter).Decode(&doc); err != nil {
		return nil, mapMongoError(err)
	}
	return docToAlertContact(doc)
}

type alertContactDoc struct {
	ID        string `bson:"_id"`
	UserLogin string `bson:"user_login"`
	Type      string `bson:"type"`
	Name      string `bson:"name"`
	Config    string `bson:"config"`
	IsActive  bool   `bson:"is_active"`
}

func docToAlertContact(d alertContactDoc) (*alert.Contact, error) {
	contactType := alert.ContactType(d.Type)
	cfg, err := unmarshalContactConfig(contactType, []byte(d.Config))
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(d.ID)
	if err != nil {
		return nil, err
	}
	return &alert.Contact{
		ID:       id,
		User:     &user.User{Login: d.UserLogin},
		Name:     d.Name,
		Type:     contactType,
		Config:   cfg,
		IsActive: d.IsActive,
	}, nil
}
