package demo

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/vlonebara/photomap/db"
	"golang.org/x/crypto/bcrypt"
)

// these credentials belong only to explicitly seeded demonstration accounts
const Password = "DemoPass123!"

//go:embed photos.json
var archiveJSON []byte

type archivePhoto struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Place       string     `json:"place"`
	Coordinates [2]float64 `json:"coordinates"`
	Year        int64      `json:"year"`
	EndYear     *int64     `json:"endYear"`
	SourceName  string     `json:"sourceName"`
	SourceURL   string     `json:"sourceUrl"`
	Stars       int64      `json:"stars"`
	ImageKey    string     `json:"imageKey"`
}

type archive struct {
	AuthorName string         `json:"authorName"`
	Photos     []archivePhoto `json:"photos"`
}

type account struct {
	Name  string
	Email string
	Role  string
}

var accounts = []account{
	{Name: "Демо-модератор", Email: "moderator@demo.photomap.invalid", Role: "moderator"},
	{Name: "Алиса", Email: "alice@demo.photomap.invalid", Role: "user"},
	{Name: "Борис", Email: "boris@demo.photomap.invalid", Role: "user"},
	{Name: "Вера", Email: "vera@demo.photomap.invalid", Role: "user"},
}

// importing fixtures explicitly without replacing existing users or photos
func Seed(ctx context.Context, store *db.Store, mediaDirectory string) error {
	var data archive
	if err := json.Unmarshal(archiveJSON, &data); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return store.WithTx(ctx, func(q *db.Queries) error {
		ids, err := seedAccounts(ctx, q, string(hash))
		if err != nil {
			return err
		}

		author, err := disabledUser(ctx, q, data.AuthorName, "archive@demo.photomap.invalid")
		if err != nil {
			return err
		}

		for _, photo := range data.Photos {
			if err := seedArchivePhoto(ctx, q, photo, author, mediaDirectory); err != nil {
				return err
			}
		}

		return seedLifecycle(ctx, q, data.Photos, ids, mediaDirectory)
	})
}

func seedAccounts(ctx context.Context, q *db.Queries, hash string) ([]int64, error) {
	ids := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		err := q.EnsureDemoAccount(ctx, db.EnsureDemoAccountParams{
			Name:         account.Name,
			Email:        account.Email,
			PasswordHash: hash,
			Role:         account.Role,
		})
		if err != nil {
			return nil, err
		}

		user, err := q.GetUserByEmail(ctx, account.Email)
		if err != nil {
			return nil, err
		}

		if user.Role != account.Role || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(Password)) != nil {
			return nil, fmt.Errorf("demo account already belongs to another user: %s", account.Email)
		}

		ids = append(ids, user.ID)
	}

	return ids, nil
}

func disabledUser(ctx context.Context, q *db.Queries, name, email string) (int64, error) {
	err := q.EnsureDemoUser(ctx, db.EnsureDemoUserParams{Name: name, Email: email})
	if err != nil {
		return 0, err
	}

	user, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		return 0, err
	}

	if user.PasswordHash != "!demo-login-disabled" {
		return 0, fmt.Errorf("demo identity already belongs to another user: %s", email)
	}

	return user.ID, nil
}
