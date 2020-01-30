package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	user "github.com/retgits/acme-serverless-user"
	"github.com/retgits/acme-serverless-user/internal/datastore/dynamodb"
)

func main() {
	os.Setenv("REGION", "us-west-2")
	os.Setenv("TABLE", "User")

	data, err := ioutil.ReadFile("./data.json")
	if err != nil {
		log.Println(err)
	}

	var users []user.User

	err = json.Unmarshal(data, &users)
	if err != nil {
		log.Println(err)
	}

	dynamoStore := dynamodb.New()

	for _, usr := range users {
		err = dynamoStore.AddUser(usr)
		if err != nil {
			log.Println(err)
		}
	}
}
