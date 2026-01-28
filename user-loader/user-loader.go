// This program performs the following tasks:
// 1. Generates a list of 100 unique users with a unique name, UUID and a tier type.
// 2. Writes the generated users to a file named `users.json`.
// 3. Upserts the generated users to a Couchbase cluster.
// 4. Performs an N1QL query to count the number of users successfully upserted into the keyspace.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Update this to your Couchbase server cluster details
const defaultConnectionString = "localhost:12000"
const bucketName = "my-llm"
const scopeName = "users"
const collectionName = "accounts"
const defaultUsername = "Administrator"
const defaultPassword = "asdasd"

// Update this to the path where you want to store the generated `users.json` file
var outputFilePath = getEnvOrDefault(
	"USERS_JSON_PATH",
	filepath.Join("/", "Users", "rishitchaudhary", "Dev", "roughpad", "users.json"),
)

func getEnvOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// TierType represents the type of tier a user has
// The TierType determines the rate limit for the user
type TierType string

const (
	Bronze   TierType = "Bronze"
	Silver   TierType = "Silver"
	Gold     TierType = "Gold"
	Platinum TierType = "Platinum"
)

type User struct {
	Name   string   `json:"name"`
	UserID string   `json:"user_id"`
	Tier   TierType `json:"tier"`
}

// generateUniqueNames generates a list of unique names of the format `User-<number>`
func generateUniqueNames(count int) []string {
	names := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		names = append(names, fmt.Sprintf("User-%d", i))
	}
	return names
}

// generateUsers generates a list of `User` objects with the given slices of `names` and `packages`
func generateUsers(names []string, tiers []TierType) []User {
	var users []User
	c := cases.Title(language.BritishEnglish)
	for i := 0; i < len(names); i++ {
		name := names[i]
		user := User{
			Name:   c.String(name),
			UserID: uuid.New().String(),
			Tier:   tiers[i%len(tiers)],
		}
		users = append(users, user)
	}
	return users
}

func main() {
	names := generateUniqueNames(100)
	tiers := []TierType{Bronze, Silver, Gold, Platinum}

	users := generateUsers(names, tiers)

	// Write the data in the variable `users` to a file with the path `outputFilePath`
	if err := writeToFile(users, outputFilePath); err != nil {
		panic(err)
	}

	for _, user := range users {
		fmt.Printf("Name: %s, UserID: %s, Tier: %s\n", user.Name, user.UserID, user.Tier)
	}

	if err := sendToCouchbaseCluster(users); err != nil {
		panic(err)
	}
}

// sendToCouchbaseCluster sends the data in the variable `users` to the configured Couchbase cluster
func sendToCouchbaseCluster(users []User) error {
	// Uncomment following line to enable logging
	// gocb.SetLogger(gocb.VerboseStdioLogger())

	// For a secure cluster connection, use `couchbases://<your-cluster-ip>:<data-service-port-number>` instead.
	connectionString := getEnvOrDefault("CB_CONNECTION_STRING", defaultConnectionString)
	username := getEnvOrDefault("CB_USERNAME", defaultUsername)
	password := getEnvOrDefault("CB_PASSWORD", defaultPassword)

	cluster, err := gocb.Connect("couchbase://"+connectionString, gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: username,
			Password: password,
		},
	})
	if err != nil {
		return err
	}
	defer cluster.Close(nil)

	bucket := cluster.Bucket(bucketName)

	err = bucket.WaitUntilReady(5*time.Second, nil)
	if err != nil {
		return err
	}
	fmt.Println("Connected to bucket")

	col := bucket.Scope(scopeName).Collection(collectionName)
	fmt.Println("Got GoCB collection object")

	// UserDocument represents the structure of the document to be stored in the Couchbase bucket
	type UserDocument struct {
		Name string   `json:"name"`
		Tier TierType `json:"tier"`
	}

	// Create and store a Document for each `user` in the `users` slice
	for _, user := range users {
		_, err = col.Upsert(user.UserID,
			UserDocument{
				Name: user.Name,
				Tier: user.Tier,
			}, nil)
		if err != nil {
			return err
		}
	}
	fmt.Println("Inserted all user documents")

	// Perform a N1QL Query
	inventoryScope := bucket.Scope(scopeName)
	queryResult, err := inventoryScope.Query(
		fmt.Sprintf("SELECT count(*) FROM %s", collectionName),
		&gocb.QueryOptions{Adhoc: true},
	)
	if err != nil {
		return err
	}

	// Print each found Row
	for queryResult.Next() {
		var result interface{}
		err := queryResult.Row(&result)
		if err != nil {
			return err
		}
		fmt.Printf("The number of users in the keyspace are %v\n", result)
	}

	if err := queryResult.Err(); err != nil {
		return err
	}
	return nil
}

// writeToFile writes the data in the variable `users` to a file in the path `filePath` in JSON format
func writeToFile(users []User, filePath string) error {
	value, err := json.Marshal(users)
	if err != nil {
		return err
	}
	err = os.WriteFile(filePath, value, 0666)
	if err != nil {
		return err
	}
	return nil
}
