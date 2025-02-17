package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Measurement struct {
	Id          string  `json:"id"`
	Temperature float64 `json:"temperature"`
	Humidity    int     `json:"humidity"`
	Voltage     float64 `json:"voltage"`
	Timestamp   int     `json:"timestamp"`
}

type Creds struct {
	ConnectionString string `json:"connectionString"`
	BucketName       string `json:"bucketName"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	Scope            string `json:"scope"`
	Collection       string `json:"collection"`
}

func natsCredsPath() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	exPath := filepath.Dir(ex)
	return filepath.Join(exPath, "nats.creds"), nil
}

func loadCreds() (*Creds, error) {
	ex, err := os.Executable()
	if err != nil {
		return &Creds{}, err
	}
	exPath := filepath.Dir(ex)
	jsonPath := filepath.Join(exPath, "credentials.json")
	b, err := os.ReadFile(jsonPath)
	if err != nil {
		return &Creds{}, err
	}
	var creds Creds
	err = json.Unmarshal(b, &creds)
	if err != nil {
		return &Creds{}, err
	}
	return &creds, nil
}

func handleMsg(msg jetstream.Msg) {
	var data Measurement
	err := json.Unmarshal(msg.Data(), &data)
	if err != nil {
		log.Fatal(err)
	}

	newId := fmt.Sprintf("%s-%v", strings.ReplaceAll(data.Id, ":", ""), data.Timestamp)
	_, err = col.Upsert(newId, data, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Inserted: ", newId)

	msg.Ack()
}

var col *gocb.Collection

func main() {
	log.Println("Starting application...")

	log.Println("Loading couchbase credentials...")
	creds, err := loadCreds()
	if err != nil {
		log.Fatal(err)
	}

	options := gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: creds.Username,
			Password: creds.Password,
		},
	}
	if err := options.ApplyProfile(gocb.ClusterConfigProfileWanDevelopment); err != nil {
		log.Fatal(err)
	}

	log.Println("Connecting to couchbase...")
	cluster, err := gocb.Connect("couchbases://"+creds.ConnectionString, options)
	if err != nil {
		log.Fatal(err)
	}

	bucket := cluster.Bucket(creds.BucketName)
	err = bucket.WaitUntilReady(10*time.Second, nil)
	if err != nil {
		log.Fatal(err)
	}
	col = bucket.Scope(creds.Scope).Collection(creds.Collection)

	log.Println("Connecting to NATS...")
	natsCreds, err := natsCredsPath()
	if err != nil {
		log.Fatal(err)
	}
	nc, err := nats.Connect("tls://connect.ngs.global", nats.UserCredentials(natsCreds))
	if err != nil {
		log.Fatal(err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	stream, err := js.Stream(ctx, "sensors")
	if err != nil {
		log.Fatal(err)
	}

	consumer, err := stream.Consumer(ctx, "consumer")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Starting consuming from NATS...")
	cc, err := consumer.Consume(handleMsg)
	if err != nil {
		log.Fatal(err)
	}
	defer cc.Stop()

	c := make(chan bool)
	<-c // sleep on channel infinitely
}
