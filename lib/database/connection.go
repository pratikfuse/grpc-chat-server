package database

import (
	"context"
)

// var Db *mongo.Database

var Ctx = context.Background()

func CreateConnection() error {
	// connection, err := mongo.Connect(Ctx, options.Client().ApplyURI("mongodb://pushy_rw:hs728hjkHk19030lkhd@localhost:27017/?authSource=admin"))

	// fmt.Println(connection, err)

	// if err != nil {
	// 	return err
	// }

	// Db = connection.Database("chatdb")
	// fmt.Println("connected to database")

	return nil
}
