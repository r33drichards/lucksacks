package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"
)

var (
	AWSRegion = "us-east-1"
)

type nameDays struct {
	streakEntry
}

func (nd nameDays) Days() int {
	start := nd.Date
	end := time.Now()
	return int(end.Sub(start).Hours() / 24)
}

func streak(s slack.SlashCommand) (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.Region = AWSRegion
	})
	text := strings.TrimSpace(s.Text)
	streakTable := StreakTable{
		TableName:      "streaks",
		DynamoDbClient: client,
	}

	// todo should prob do this somewhere else but for now it's fine
	_, _ = streakTable.CreateTable()

	// if text is empty, return list of all streaks
	if text == "" {
		strks, err := streakTable.ListStreaks(s.UserID)
		if err != nil {
			return "", errors.Wrap(err, "failed to list streaks")
		}
		msg := "Streaks for " + s.UserName + ":\n"
		for _, strk := range strks {
			msg += strk.Name() + ": " + fmt.Sprintf("%d", nameDays{streakEntry: strk}.Days()) + " days\n"
		}
		return msg, nil
	}

	streakEntry := streakEntry{
		Date:         time.Now(),
		SlashCommand: s,
	}

	// if text begins with "delete", delete streak by name
	if strings.HasPrefix(text, "delete") {
		err = streakTable.DeleteStreak(streakEntry)
		if err != nil {
			return "", errors.Wrap(err, "failed to delete streak")
		}
		return "Deleted streak " + streakEntry.Name(), nil
	}
	// if text begins with "add", add streak by name
	if strings.HasPrefix(text, "add") {
		err = streakTable.AddStreak(streakEntry)
		if err != nil {
			return "", errors.Wrap(err, "failed to add streak")
		}
		return "Added streak " + streakEntry.Name(), nil
	}

	return "", nil
}

type streakEntry struct {
	Date         time.Time
	SlashCommand slack.SlashCommand
}

func (se *streakEntry) Name() string {
	// remove first word from text and return the rest
	text := strings.TrimSpace(se.SlashCommand.Text)
	words := strings.Split(text, " ")
	if len(words) > 1 {
		return strings.Join(words[1:], " ")
	}
	return ""
}

type StreakTable struct {
	DynamoDbClient *dynamodb.Client
	TableName      string
}

// snippet-end:[gov2.dynamodb.TableBasics.struct]

// snippet-start:[gov2.dynamodb.DescribeTable]

// CreatTable creates a DynamoDB table with primary key pk and sort key sk.
func (basics StreakTable) CreateTable() (*types.TableDescription, error) {
	var tableDesc *types.TableDescription
	table, err := basics.DynamoDbClient.CreateTable(context.TODO(),
		&dynamodb.CreateTableInput{
			TableName: aws.String(basics.TableName),
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String("pk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("sk"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("pk"),
					KeyType:       types.KeyTypeHash,
				},
				{
					AttributeName: aws.String("sk"),
					KeyType:       types.KeyTypeRange,
				},
			},
			BillingMode: types.BillingModePayPerRequest,
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create table")
	} else {
		waiter := dynamodb.NewTableExistsWaiter(basics.DynamoDbClient)
		err = waiter.Wait(context.TODO(), &dynamodb.DescribeTableInput{
			TableName: aws.String(basics.TableName)}, 5*time.Minute)
		if err != nil {
			return nil, errors.Wrap(err, "failed to wait for table to exist")
		}
		tableDesc = table.TableDescription
	}
	return tableDesc, err
}

// AddMovie adds a movie the DynamoDB table.
func (basics StreakTable) AddStreak(streak streakEntry) error {
	// item, err := attributevalue.MarshalMap(streak)
	// if err != nil {
	// 	panic(err)
	// }
	item := map[string]types.AttributeValue{
		"pk":   &types.AttributeValueMemberS{Value: streak.SlashCommand.UserID},
		"sk":   &types.AttributeValueMemberS{Value: streak.Name()},
		"date": &types.AttributeValueMemberS{Value: streak.Date.Format("2006-01-02")},
		"text": &types.AttributeValueMemberS{Value: streak.SlashCommand.Text},
	}

	_, err := basics.DynamoDbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(basics.TableName), Item: item,
	})
	if err != nil {
		log.WithFields(
			log.Fields{
				"error": err,
				"item":  item,
				"streak": streak,
			},
		).Error("failed to add streak")
		return errors.Wrap(err, "failed to add streak")
	}
	return err
}

func (basics StreakTable) DeleteStreak(streak streakEntry) error {
	_, err := basics.DynamoDbClient.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String(basics.TableName),
		Key:       map[string]types.AttributeValue{"pk": &types.AttributeValueMemberS{Value: streak.SlashCommand.UserID}, "sk": &types.AttributeValueMemberS{Value: streak.Name()}},
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete streak")
	}
	return err
}

// list streaks for a user
func (basics StreakTable) ListStreaks(user string) ([]streakEntry, error) {
	result, err := basics.DynamoDbClient.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String(basics.TableName),
		KeyConditionExpression: aws.String("pk = :v1"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":v1": &types.AttributeValueMemberS{Value: user},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to query table")
	}
	log.Printf("result: %v", result)
	var streaks []streakEntry
	for _, item := range result.Items {
		streak := streakEntry{}
		streak.SlashCommand.UserID = item["pk"].(*types.AttributeValueMemberS).Value
		streak.Date, _ = time.Parse("2006-01-02", item["date"].(*types.AttributeValueMemberS).Value)
		streak.SlashCommand.Text = item["text"].(*types.AttributeValueMemberS).Value
		streaks = append(streaks, streak)
	}
	return streaks, nil
}
