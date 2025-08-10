package main

import (
	"go.mongodb.org/mongo-driver/mongo"
	"context"
)

// MongoDBで管理するGCEインスタンス情報
type GCEInstanceConfig struct {
	GuildID      string `bson:"guild_id"`
	CommandName  string `bson:"command_name"`
	Project      string `bson:"project"`
	Zone         string `bson:"zone"`
	InstanceName string `bson:"instance_name"`
}

func getGCEConfigFromMongo(client *mongo.Client, guildID, commandName string) (*GCEInstanceConfig, error) {
	collection := client.Database("servermanager").Collection("gce_configs")
	filter := map[string]interface{}{
		"guild_id":     guildID,
		"command_name": commandName,
	}
	var config GCEInstanceConfig
	err := collection.FindOne(context.Background(), filter).Decode(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}