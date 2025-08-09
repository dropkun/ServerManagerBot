package main

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"strings"
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

func main() {
	_ = godotenv.Load()
	token := os.Getenv("DISCORD_TOKEN")
	mongoURI := os.Getenv("MONGODB_URI")
	if token == "" {
		fmt.Println("環境変数 DISCORD_TOKEN が見つかりません。")
		os.Exit(1)
	}
	if mongoURI == "" {
		fmt.Println("環境変数 MONGODB_URI が見つかりません。")
		os.Exit(1)
	}

	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		fmt.Println("MongoDB接続失敗:", err)
		return
	}
	defer mongoClient.Disconnect(context.Background())

	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Discordセッション作成失敗:", err)
		return
	}

	sess.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}
		guildID := i.GuildID
		name := i.ApplicationCommandData().Name
		var gceConfig *GCEInstanceConfig
		gceConfig, err = getGCEConfigFromMongo(mongoClient, guildID, name)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "GCE設定が見つかりません: " + err.Error()},
			})
			return
		}
		instance := NewInstanceController(gceConfig.Project, gceConfig.Zone, gceConfig.InstanceName)

		action := ""
		for _, opt := range i.ApplicationCommandData().Options {
			if opt.Name == "action" {
				action = strings.ToLower(opt.StringValue())
			}
		}
		if action == "start" {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Starting a " + name + " server..."},
			})
			go func() {
				result := instance.Start()
				ip := instance.GetExternalIP()
				msg := result + "\n" + ip
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
			}()
		} else if action == "stop" {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Stopping a " + name + " server..."},
			})
			go func() {
				result := instance.Stop()
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &result})
			}()
		} else {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "未知のサブコマンドです"},
			})
		}
	})

	err = sess.Open()
	if err != nil {
		fmt.Println("Bot起動失敗:", err)
		return
	}
	fmt.Println("Bot起動完了")

	_, err = sess.ApplicationCommandCreate(sess.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Ping pong!",
	})
	_, err = sess.ApplicationCommandCreate(sess.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "mc",
		Description: "Minecraft server controll",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "action",
				Description: "start or stop",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "start", Value: "start"},
					{Name: "stop", Value: "stop"},
				},
			},
		},
	})
	select {}
}
