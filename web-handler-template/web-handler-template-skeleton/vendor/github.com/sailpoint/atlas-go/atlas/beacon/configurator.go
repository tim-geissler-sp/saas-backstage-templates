// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package beacon

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/sailpoint/atlas-go/atlas/config"
)

var (
	tenantServiceConfigTable = aws.String("tenant_service_config")
	serviceConfigTable       = aws.String("service_config")
)

// DynamoConfigurator is the default Configurator implementation that reads configuration values from DynamoDB.
type DynamoConfigurator struct {
	dynamo *dynamodb.DynamoDB
}

// NewDynamoConfigurator constructs a new DynamoConfigurator, using us-east-1 as the default region.
func NewDynamoConfigurator() *DynamoConfigurator {
	c := &DynamoConfigurator{}
	c.dynamo = dynamodb.New(config.GlobalAwsSession(), aws.NewConfig().WithRegion("us-east-1"))

	return c
}

// FindByTenantAndService Looks up the configuration in dynamo for the specified tenant/service.
func (c *DynamoConfigurator) FindByTenantAndService(tenantID TenantID, serviceID ServiceID) (*Configuration, error) {
	name, err := c.getServiceConfigName(tenantID, serviceID)
	if err != nil {
		return nil, err
	}

	return c.getServiceConfig(name)
}

// getServiceConfigName reads the name of the service_config entry for the specified tenant/service.
func (c *DynamoConfigurator) getServiceConfigName(tenantID TenantID, serviceID ServiceID) (string, error) {
	input := &dynamodb.GetItemInput{
		TableName: tenantServiceConfigTable,
		Key: map[string]*dynamodb.AttributeValue{
			"tenant":  {S: aws.String(string(tenantID))},
			"service": {S: aws.String(string(serviceID))},
		},
	}

	result, err := c.dynamo.GetItem(input)
	if err != nil {
		return "", fmt.Errorf("get service config for %s - %s: %w", tenantID, serviceID, err)
	}

	if result.Item == nil {
		return "", fmt.Errorf("no service config for %s - %s", tenantID, serviceID)
	}

	value := result.Item["service_config_name"]
	if value == nil || value.S == nil || *value.S == "" {
		return "", fmt.Errorf("no service config for %s - %s", tenantID, serviceID)
	}

	return *value.S, nil
}

// getServiceConfig reads configuration with the specified name from dynamo.
func (c *DynamoConfigurator) getServiceConfig(name string) (*Configuration, error) {
	input := &dynamodb.GetItemInput{
		TableName: serviceConfigTable,
		Key: map[string]*dynamodb.AttributeValue{
			"name": {S: aws.String(name)},
		},
	}

	result, err := c.dynamo.GetItem(input)
	if err != nil {
		return nil, fmt.Errorf("get service config %s: %w", name, err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("no service config for %s", name)
	}

	config := make(Configuration)
	for k, v := range result.Item {
		if k == "name" {
			continue
		}

		if v.S != nil {
			config[k] = *v.S
		}
	}

	return &config, nil
}
