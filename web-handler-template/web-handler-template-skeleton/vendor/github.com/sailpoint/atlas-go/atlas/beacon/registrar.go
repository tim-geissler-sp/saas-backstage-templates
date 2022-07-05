// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package beacon

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
	"github.com/sailpoint/atlas-go/atlas/config"
	"github.com/sailpoint/atlas-go/atlas/dynamoutil"
)

var (
	ErrNotImplemented = errors.New("not implemented")
)

var (
	registryTable       = aws.String("beacon_registry")
	connectionListTable = aws.String("connection_list")
	tenantIndex         = aws.String("tenant_id-index")
	tenantServiceIndex  = aws.String("tenant_id-service_id-index")
	serviceIndex        = aws.String("service_id-index")
)

// DynamoRegistrar is the default Registrar implementation that uses DynamoDB.
type DynamoRegistrar struct {
	dynamo *dynamodb.DynamoDB
}

// NewDynamoRegistrar constructs a new DynamoRegistrar, using us-east-1 as the AWS region.
func NewDynamoRegistrar() *DynamoRegistrar {
	r := &DynamoRegistrar{}
	r.dynamo = dynamodb.New(config.GlobalAwsSession(), aws.NewConfig().WithRegion("us-east-1"))

	return r
}

// Register creates a new Registration in Dynamo.
func (r *DynamoRegistrar) Register(request RegistrationRequest) (*Registration, error) {
	connection, err := r.getConnection(request.ConnectionID, request.Port)
	if err != nil {
		return nil, err
	}

	registration := &Registration{
		ID:         newRegistrationID(),
		Created:    time.Now().UTC(),
		TenantID:   request.TenantID,
		ServiceID:  request.ServiceID,
		Hostname:   request.Hostname,
		Connection: connection,
	}

	item := toItem(registration)
	item["expiration"] = dynamoutil.NumberAttribute(time.Now().Add(2 * time.Minute).Unix())

	_, err = r.dynamo.PutItem(&dynamodb.PutItemInput{
		TableName: registryTable,
		Item:      item,
	})

	if err != nil {
		return nil, err
	}

	return registration, nil
}

// Heartbeat updates the expiration of a Registration
func (r *DynamoRegistrar) Heartbeat(registrationID RegistrationID) (bool, error) {
	result, err := r.dynamo.GetItem(&dynamodb.GetItemInput{
		TableName: registryTable,
		Key: map[string]*dynamodb.AttributeValue{
			"id": dynamoutil.StringAttribute(string(registrationID)),
		},
	})

	if err != nil {
		return false, err
	}

	item := result.Item
	if item == nil {
		return false, nil
	}

	expired, err := isExpired(item)
	if err != nil {
		return false, err
	}

	if expired {
		return false, nil
	}

	item["expiration"] = dynamoutil.NumberAttribute(time.Now().Add(2 * time.Minute).Unix())

	_, err = r.dynamo.PutItem(&dynamodb.PutItemInput{
		TableName: registryTable,
		Item:      item,
	})

	if err != nil {
		return false, err
	}

	return true, nil
}

// Cancel deletes a Registration
func (r *DynamoRegistrar) Cancel(registrationID RegistrationID) error {
	_, err := r.dynamo.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: registryTable,
		Key: map[string]*dynamodb.AttributeValue{
			"id": dynamoutil.StringAttribute(string(registrationID)),
		},
	})

	if err != nil {
		return fmt.Errorf("cancel registration '%s': %w", registrationID, err)
	}

	return nil
}

// FindAllByTenant returns a list of all Registrations for the specified tenant.
func (r *DynamoRegistrar) FindAllByTenant(tenantID TenantID) ([]*Registration, error) {
	return nil, ErrNotImplemented
}

// FindAllByService returns a list of all Registrations for the specified service.
func (r *DynamoRegistrar) FindAllByService(serviceID ServiceID) ([]*Registration, error) {
	return nil, ErrNotImplemented
}

// FindByTenantAndService finds the registration for the specified tenant/service. Nil is returned if no
// registration exists.
func (r *DynamoRegistrar) FindByTenantAndService(tenantID TenantID, serviceID ServiceID) (*Registration, error) {

	expressionAttributeValues := make(map[string]*dynamodb.AttributeValue)
	expressionAttributeValues[":tenantID"] = dynamoutil.StringAttribute(string(tenantID))
	expressionAttributeValues[":serviceID"] = dynamoutil.StringAttribute(string(serviceID))

	out, err := r.dynamo.Query(&dynamodb.QueryInput{
		TableName:                 registryTable,
		IndexName:                 tenantServiceIndex,
		KeyConditionExpression:    aws.String("tenant_id = :tenantID AND service_id = :serviceID"),
		ExpressionAttributeValues: expressionAttributeValues,
	})

	if err != nil {
		return nil, err
	}

	registrations, err := fromItems(out.Items)
	if len(registrations) == 0 {
		return nil, err
	}

	return registrations[0], nil
}

// getConnection gets the connection string for the specified connection name and port.
func (r *DynamoRegistrar) getConnection(connectionID ConnectionID, port int) (string, error) {
	result, err := r.dynamo.GetItem(&dynamodb.GetItemInput{
		TableName: connectionListTable,
		Key: map[string]*dynamodb.AttributeValue{
			"name": dynamoutil.StringAttribute(string(connectionID)),
		},
	})

	if err != nil {
		return "", err
	}

	connection := ""
	if port == 443 {
		connection = dynamoutil.GetString(result.Item["connection"])
	} else {
		connection = dynamoutil.GetString(result.Item[strconv.Itoa(port)])
	}

	if connection == "" {
		return "", fmt.Errorf("no port defined in dynamo connection")
	}

	return connection, nil
}

// newRegistrationID constructs a new randomly-generated RegistrationID.
func newRegistrationID() RegistrationID {
	value := uuid.New().String()
	value = strings.ReplaceAll(value, "-", "")

	return RegistrationID(value)
}

// toItem converts a registration to a dynamo item.
func toItem(r *Registration) map[string]*dynamodb.AttributeValue {
	return map[string]*dynamodb.AttributeValue{
		"id":         dynamoutil.StringAttribute(string(r.ID)),
		"created":    dynamoutil.TimeAttribute(r.Created),
		"tenant_id":  dynamoutil.StringAttribute(string(r.TenantID)),
		"service_id": dynamoutil.StringAttribute(string(r.ServiceID)),
		"hostname":   dynamoutil.StringAttribute(r.Hostname),
		"connection": dynamoutil.StringAttribute(r.Connection),
	}
}

// fromItems converts dynamo items to registrations
func fromItems(items []map[string]*dynamodb.AttributeValue) ([]*Registration, error) {
	var registrations []*Registration

	for _, item := range items {
		registration, err := fromItem(item)
		if err != nil {
			return nil, err
		}

		registrations = append(registrations, registration)
	}

	return registrations, nil
}

// fromItem converts a dynamo item to a registration
func fromItem(item map[string]*dynamodb.AttributeValue) (*Registration, error) {
	if item == nil {
		return nil, nil
	}

	created, err := dynamoutil.GetTime(item["created"])
	if err != nil {
		return nil, err
	}

	registration := &Registration{
		ID:         RegistrationID(dynamoutil.GetString(item["id"])),
		Created:    created,
		TenantID:   TenantID(dynamoutil.GetString(item["tenant_id"])),
		ServiceID:  ServiceID(dynamoutil.GetString(item["service_id"])),
		Hostname:   dynamoutil.GetString(item["hostname"]),
		Connection: dynamoutil.GetString(item["connection"]),
	}

	return registration, nil
}

// isExpired gets whether or not the specified registry item is expired.
func isExpired(item map[string]*dynamodb.AttributeValue) (bool, error) {
	expirationValue := item["expiration"]

	if expirationValue.N == nil {
		return false, fmt.Errorf("expiration value is invalid")
	}

	expirationNumber, err := strconv.ParseInt(*expirationValue.N, 10, 64)
	if err != nil {
		return false, err
	}

	expiration := time.Unix(expirationNumber, 0)
	return expiration.Before(time.Now()), nil
}
