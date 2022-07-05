// Copyright (c) 2022, SailPoint Technologies, Inc. All rights reserved.
package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
)

func TestPollingNextResultsForNonExistentInvocation(t *testing.T) {
	e := httpexpect.New(t, *orgUrl)
	e = e.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+token)
	})

	result := e.POST("/sp-connect/invocations/abcd/next-result").
		WithJSON(map[string]interface{}{
			"limit":   1,
			"timeout": "1s",
		}).
		Expect().
		Status(http.StatusOK).JSON().Object()

	result.Equal(map[string]interface{}{
		"done":    false,
		"context": nil,
		"output":  []interface{}{},
	})
}

func TestInvokeCommandOnInternalConnector(t *testing.T) {
	e := httpexpect.New(t, *orgUrl)
	e = e.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+token)
	})

	defer cleanupInternalConnectorInstances(e)

	// Create connector instance
	create := e.POST("/sp-connect/connector-instances").
		WithJSON(map[string]interface{}{
			"name":            "sp-connect karate test",
			"connectorSpecId": "internal",
			"config":          map[string]interface{}{},
		}).
		Expect().
		Status(http.StatusOK).JSON().Object()

	// Invoke commands
	testConnection := invokeCommand(e, create.Value("id").String().Raw(), "std:test-connection", map[string]interface{}{},
		map[string]interface{}{
			"error": "",
			"done":  true,
			"output": []interface{}{
				map[string]interface{}{},
			},
		})

	authenticate := invokeCommand(e, create.Value("id").String().Raw(), "std:authenticate",
		map[string]interface{}{
			"username": "karate",
			"password": "apple",
		},
		map[string]interface{}{
			"error": "",
			"done":  true,
			"output": []interface{}{
				map[string]interface{}{
					"identity":   "john.doe",
					"uuid":       "1234",
					"attributes": map[string]interface{}{},
				},
			},
		})

	accountList := invokeCommand(e, create.Value("id").String().Raw(), "std:account:list", map[string]interface{}{},
		map[string]interface{}{
			"error": "",
			"done":  true,
			"output": []interface{}{
				map[string]interface{}{
					"identity":   "john.doe",
					"uuid":       "1234",
					"attributes": map[string]interface{}{},
				},
			},
		})

	accountRead := invokeCommand(e, create.Value("id").String().Raw(), "std:account:read",
		map[string]interface{}{
			"identity": "john.doe",
		},
		map[string]interface{}{
			"error": "",
			"done":  true,
			"output": []interface{}{
				map[string]interface{}{
					"identity":   "john.doe",
					"uuid":       "1234",
					"attributes": map[string]interface{}{},
				},
			},
		})

	accountCreate := invokeCommand(e, create.Value("id").String().Raw(), "std:account:create",
		map[string]interface{}{
			"identity": "john.doe",
			"attributes": map[string]interface{}{
				"first": "john",
				"last":  "doe",
				"email": "john.doe@example.com",
			},
		},
		map[string]interface{}{
			"error": "",
			"done":  true,
			"output": []interface{}{
				map[string]interface{}{
					"identity": "john.doe",
					"uuid":     "1234",
					"attributes": map[string]interface{}{
						"first": "john",
						"last":  "doe",
						"email": "john.doe@example.com",
					},
				},
			},
		})

	accountUpdate := invokeCommand(e, create.Value("id").String().Raw(), "std:account:update",
		map[string]interface{}{
			"identity": "john.doe",
			"changes": []interface{}{
				map[string]interface{}{
					"op":        "Add",
					"attribute": "groups",
					"value":     []interface{}{"Group1", "Group2"},
				},
				map[string]interface{}{
					"op":        "Set",
					"attribute": "phone",
					"value":     2223334444,
				},
				map[string]interface{}{
					"op":        "Remove",
					"attribute": "location",
				},
			},
		},
		map[string]interface{}{
			"error": "",
			"done":  true,
			"output": []interface{}{
				map[string]interface{}{
					"identity":   "john.doe",
					"uuid":       "1234",
					"attributes": map[string]interface{}{},
				},
			},
		})

	accountDelete := invokeCommand(e, create.Value("id").String().Raw(), "std:account:delete",
		map[string]interface{}{
			"identity": "john.doe",
		},
		map[string]interface{}{
			"error":  "",
			"done":   true,
			"output": []interface{}{map[string]interface{}{}},
		})

	entitlementList := invokeCommand(e, create.Value("id").String().Raw(), "std:entitlement:list", map[string]interface{}{},
		map[string]interface{}{
			"error": "",
			"done":  true,
			"output": []interface{}{
				map[string]interface{}{
					"identity":   "deparment1",
					"uuid":       "1234",
					"attributes": map[string]interface{}{},
				},
				map[string]interface{}{
					"identity":   "deparment2",
					"uuid":       "5678",
					"attributes": map[string]interface{}{},
				},
			},
		})
	entitlementRead := invokeCommand(e, create.Value("id").String().Raw(), "std:entitlement:read",
		map[string]interface{}{
			"identity": "deparment1",
		},
		map[string]interface{}{
			"error": "",
			"done":  true,
			"output": []interface{}{
				map[string]interface{}{
					"identity":   "deparment1",
					"uuid":       "1234",
					"attributes": map[string]interface{}{},
				},
			},
		})

	readDisabledAccount := invokeCommand(e, create.Value("id").String().Raw(), "std:account:read",
		map[string]interface{}{
			"identity": "john.doe",
		},
		map[string]interface{}{
			"error": "",
			"done":  true,
			"output": []interface{}{
				map[string]interface{}{
					"identity":   "john.doe",
					"uuid":       "1234",
					"disabled":   true,
					"locked":     false,
					"attributes": map[string]interface{}{},
				},
			},
		})

	readNotExistedAccount := invokeCommand(e, create.Value("id").String().Raw(), "std:account:read",
		map[string]interface{}{
			"identity": "john.doe",
		},
		map[string]interface{}{
			"error": "[ConnectorError] Account john.doe does not exist",
			"err": map[string]interface{}{
				"category": "ConnectorError",
				"type":     "notFound",
				"message":  "Account john.doe does not exist",
			},
			"done":   true,
			"output": []interface{}{},
		})

	// Wait for invocation to go through
	time.Sleep(3 * time.Second)

	// Validate invocation result
	validateResult(e, testConnection, []interface{}{map[string]interface{}{}})
	validateResult(e, authenticate, []interface{}{map[string]interface{}{
		"identity":   "john.doe",
		"uuid":       "1234",
		"attributes": map[string]interface{}{}}})
	validateResult(e, accountList, []interface{}{map[string]interface{}{
		"identity":   "john.doe",
		"uuid":       "1234",
		"attributes": map[string]interface{}{}}})
	validateResult(e, accountRead, []interface{}{map[string]interface{}{
		"identity":   "john.doe",
		"uuid":       "1234",
		"attributes": map[string]interface{}{}}})
	validateResult(e, accountCreate, []interface{}{map[string]interface{}{
		"identity": "john.doe",
		"uuid":     "1234",
		"attributes": map[string]interface{}{
			"first": "john",
			"last":  "doe",
			"email": "john.doe@example.com",
		}}})
	validateResult(e, accountUpdate, []interface{}{map[string]interface{}{
		"identity":   "john.doe",
		"uuid":       "1234",
		"attributes": map[string]interface{}{}}})
	validateResult(e, accountDelete, []interface{}{map[string]interface{}{}})
	validateResult(e, entitlementList, []interface{}{
		map[string]interface{}{
			"identity":   "deparment1",
			"uuid":       "1234",
			"attributes": map[string]interface{}{},
		},
		map[string]interface{}{
			"identity":   "deparment2",
			"uuid":       "5678",
			"attributes": map[string]interface{}{},
		},
	})
	validateResult(e, entitlementRead, []interface{}{
		map[string]interface{}{
			"identity":   "deparment1",
			"uuid":       "1234",
			"attributes": map[string]interface{}{},
		},
	})
	validateResult(e, readDisabledAccount, []interface{}{map[string]interface{}{
		"identity":   "john.doe",
		"uuid":       "1234",
		"disabled":   true,
		"locked":     false,
		"attributes": map[string]interface{}{}}})

	validateError(e, readNotExistedAccount, "[ConnectorError] Account john.doe does not exist", "notFound")

}

func TestInvokeInvalidCommand(t *testing.T) {
	e := httpexpect.New(t, *orgUrl)
	e = e.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+token)
	})

	defer cleanupInternalConnectorInstances(e)

	create := e.POST("/sp-connect/connector-instances").
		WithJSON(map[string]interface{}{
			"name":            "sp-connect karate test",
			"connectorSpecId": "internal",
			"config":          map[string]interface{}{},
		}).
		Expect().
		Status(http.StatusOK).JSON().Object()

	// Test connection with input
	e.POST("/sp-connect/connector-instances/" + create.Value("id").String().Raw() + "/commands").
		WithJSON(map[string]interface{}{
			"type":    "std:test-connection",
			"timeout": "10s",
			"input": map[string]interface{}{
				"identity": "john.doe",
			},
			"responseConfig": map[string]interface{}{
				"type":   "sync",
				"config": map[string]interface{}{},
			},
			"context": map[string]interface{}{},
		}).
		Expect().
		Status(http.StatusBadRequest)
}

func invokeCommand(e *httpexpect.Expect, instanceId string, cmdType string, input map[string]interface{}, response map[string]interface{}) string {
	invoke := e.POST("/sp-connect/connector-instances/" + instanceId + "/commands").
		WithJSON(map[string]interface{}{
			"type":    cmdType,
			"timeout": "10s",
			"input":   input,
			"responseConfig": map[string]interface{}{
				"type":   "sync",
				"config": map[string]interface{}{},
			},
			"context":  map[string]interface{}{},
			"response": response,
		}).
		Expect().
		Status(http.StatusOK).JSON().Object()

	invoke.Value("invocationId").String()
	invoke.Value("created").String()
	invoke.Value("expiration").String()
	invoke.Value("type").String().Equal(cmdType)
	invoke.Value("connectorInstanceId").String().Equal(instanceId)
	invoke.Value("input").Object().Equal(input)

	return invoke.Value("invocationId").String().Raw()
}

func validateResult(e *httpexpect.Expect, invovationId string, output []interface{}) {
	result := e.POST("/sp-connect/invocations/" + invovationId + "/next-result").
		WithJSON(map[string]interface{}{
			"timeout": "10s",
			"limit":   10,
		}).
		Expect().
		Status(http.StatusOK).JSON().Object()

	result.Value("done").Boolean().True()
	result.Value("context").Object().Equal(map[string]interface{}{})
	result.Value("output").Array().Equal(output)
}

func validateError(e *httpexpect.Expect, invovationId string, err string, errType string) {
	result := e.POST("/sp-connect/invocations/" + invovationId + "/next-result").
		WithJSON(map[string]interface{}{
			"timeout": "10s",
			"limit":   10,
		}).
		Expect().
		Status(http.StatusOK).JSON().Object()

	result.Value("done").Boolean().True()
	result.Value("output").Array().Equal([]interface{}{})
	result.Value("context").Null()
	result.Value("error").String().Contains(err) // Need to use contain because actual error message has request ID
	result.Value("errorType").String().Equal(errType)
}
