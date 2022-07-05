package client

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/sailpoint/atlas-go/atlas/config"
	"github.com/sailpoint/atlas-go/atlas/dynamoutil"
)

var (
	orgTableDev  = aws.String("org")
	orgTableProd = aws.String("automation_prod_test_orgs")
)

//Org data repo
type orgRepo struct {
	dynamo *dynamodb.DynamoDB
}

func newOrgRepo() *orgRepo {
	r := &orgRepo{}
	r.dynamo = dynamodb.New(config.GlobalAwsSession(), aws.NewConfig().WithRegion(config.MainRegion()))
	return r
}

type orgData struct {
	CCAPIUser string
	CCAPIKey  string
	Password  string
}

func (r *orgRepo) retrieveOrgDataProd(org string) (*orgData, error) {
	out, err := r.dynamo.Query(&dynamodb.QueryInput{
		TableName:              orgTableProd,
		KeyConditionExpression: aws.String("#org = :org"),
		ExpressionAttributeNames: map[string]*string{
			"#org": aws.String("_org"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":org": dynamoutil.StringAttribute(org),
		},
	})
	if err != nil {
		return nil, err
	}
	if len(out.Items) == 0 {
		return nil, fmt.Errorf("prod org (%q) not found (%+v)", org, out.LastEvaluatedKey)
	}
	item := out.Items[0]
	return &orgData{
		CCAPIUser: *item["API_USER"].S,
		CCAPIKey:  *item["API_KEY"].S,
		Password:  *item["password"].S,
	}, nil
}

func (r *orgRepo) retrieveOrgDataDev(org string) (*orgData, error) {
	out, err := r.dynamo.Query(&dynamodb.QueryInput{
		TableName:              orgTableDev,
		IndexName:              aws.String("_name-index"),
		KeyConditionExpression: aws.String("#name = :name"),
		ExpressionAttributeNames: map[string]*string{
			"#name": aws.String("_name"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":name": dynamoutil.StringAttribute(org),
		},
	})
	if err != nil {
		return nil, err
	}
	if len(out.Items) == 0 {
		return nil, fmt.Errorf("dev org (%q) not found (%+v)", org, out.LastEvaluatedKey)
	}
	item := out.Items[0]
	return &orgData{
		CCAPIUser: *item["cc_api_user"].S,
		CCAPIKey:  *item["cc_api_key"].S,
		Password:  "",
	}, nil
}
