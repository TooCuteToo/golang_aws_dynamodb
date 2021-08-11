// main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/bxcodec/faker/v3"
)

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(handleRequest)
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var response events.APIGatewayProxyResponse

	switch true {
	case request.HTTPMethod == "GET" && request.Path == "/products" && request.PathParameters["id"] == "":
		products, _ := NewDynamoDBRepository().FindAll()
		result, _ := json.Marshal(products)
		response = events.APIGatewayProxyResponse{Body: string(result), StatusCode: 200, Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		}}

	case request.HTTPMethod == "GET" && request.Resource == "/products/{id}" && request.PathParameters["id"] != "":
		products, _ := NewDynamoDBRepository().FindByID(request.PathParameters["id"])
		result, _ := json.Marshal(products)
		response = events.APIGatewayProxyResponse{Body: string(result), StatusCode: 200, Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		}}

	case request.HTTPMethod == "POST" && request.Path == "/products":
		for i := 1; i < 5; i++ {
			NewDynamoDBRepository().Save(&Product{
				Id:          faker.Word(),
				Name:        faker.Word(),
				Description: faker.Paragraph(),
				Price:       10 + rand.Float32()*(100-10),
				Rate:        0 + rand.Float32()*(5-0),
				Image:       fmt.Sprintf("http://lorempixel.com/200/200?%s", faker.UUIDDigit()),
			})
		}
		response = events.APIGatewayProxyResponse{Body: "Initialize DynamoDB is complete!!!", StatusCode: 200, Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		}}

	case request.HTTPMethod == "DELETE" && request.Resource == "/products/{id}" && request.PathParameters["id"] != "":
		err := NewDynamoDBRepository().Delete(request.PathParameters["id"])

		if err != nil {
			response = events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 400, Headers: map[string]string{
				"Access-Control-Allow-Origin": "*",
			}}
		}

		message, _ := json.Marshal(Message{"Done"})
		response = events.APIGatewayProxyResponse{Body: string(message), StatusCode: 200, Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		}}

	case request.HTTPMethod == "PUT" && request.Resource == "/products/{id}" && request.PathParameters["id"] != "":
		var product Product

		json.Unmarshal([]byte(request.Body), &product)
		NewDynamoDBRepository().Put(request.PathParameters["id"], product)

		message, _ := json.Marshal(Message{"Done"})
		response = events.APIGatewayProxyResponse{Body: string(message), StatusCode: 200, Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		}}

	default:
		result, _ := json.Marshal(request)
		response = events.APIGatewayProxyResponse{Body: string(result), StatusCode: 404}
	}

	return response, nil
}

type Product struct {
	Id          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float32 `json:"price"`
	Rate        float32 `json:"rate"`
	Image       string  `json:"image"`
}

type Message struct {
	Message string `json:"message"`
}

type ProductRepository interface {
	Save(post *Product) (*Product, error)
	FindAll() ([]Product, error)
	FindByID(id string) (*Product, error)
	Delete(id string) error
	Put(id string, updateProduct Product)
}

type dynamoDBRepo struct {
	tableName string
}

func NewDynamoDBRepository() ProductRepository {
	return &dynamoDBRepo{
		tableName: "Product",
	}
}

func createDynamoDBClient() *dynamodb.DynamoDB {
	// Create AWS Session
	sess, _ := session.NewSession()

	// Return DynamoDB client
	return dynamodb.New(sess, aws.NewConfig().WithRegion("ap-southeast-1"))
}

func (repo *dynamoDBRepo) Save(product *Product) (*Product, error) {
	// Get a new DynamoDB client
	dynamoDBClient := createDynamoDBClient()

	// Transforms the post to map[string]*dynamodb.AttributeValue
	attributeValue, err := dynamodbattribute.MarshalMap(product)
	if err != nil {
		return nil, err
	}

	// Create the Item Input
	item := &dynamodb.PutItemInput{
		Item:      attributeValue,
		TableName: aws.String(repo.tableName),
	}

	// Save the Item into DynamoDB
	_, err = dynamoDBClient.PutItem(item)
	if err != nil {
		return nil, err
	}

	return product, err
}

func (repo *dynamoDBRepo) FindAll() ([]Product, error) {
	// Get a new DynamoDB client
	dynamoDBClient := createDynamoDBClient()

	// Build the query input parameters
	params := &dynamodb.ScanInput{
		TableName: aws.String(repo.tableName),
	}

	// Make the DynamoDB Query API call
	result, err := dynamoDBClient.Scan(params)
	if err != nil {
		return nil, err
	}
	var products []Product = []Product{}
	for _, i := range result.Items {
		product := Product{}

		err = dynamodbattribute.UnmarshalMap(i, &product)

		if err != nil {
			panic(err)
		}

		products = append(products, product)
	}
	return products, nil
}

func (repo *dynamoDBRepo) FindByID(id string) (*Product, error) {
	// Get a new DynamoDB client
	dynamoDBClient := createDynamoDBClient()
	fmt.Print(id)

	result, err := dynamoDBClient.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(repo.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	product := Product{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &product)
	if err != nil {
		panic(err)
	}
	return &product, nil
}

func (repo *dynamoDBRepo) Delete(id string) error {
	// Get a new DynamoDB client
	dynamoDBClient := createDynamoDBClient()
	fmt.Print(id)

	_, err := dynamoDBClient.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(repo.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
	})

	if err != nil {
		return err
	}

	return nil
}

func (repo *dynamoDBRepo) Put(id string, updateProduct Product) {
	// Get a new DynamoDB client
	dynamoDBClient := createDynamoDBClient()

	result, err := dynamoDBClient.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#name": aws.String("name"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":n": {
				S: aws.String(updateProduct.Name),
			},
			":d": {
				S: aws.String(updateProduct.Description),
			},
			":pr": {
				N: aws.String(fmt.Sprintf("%f", updateProduct.Price)),
			},
		},
		TableName: aws.String(repo.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set #name = :n, description = :d, price = :pr"),
	})

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(result)
}
