// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package topics

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/sailpoint/atlas-go/atlas/event"
)

// IdnTopic is an enumeration of IdentityNow topics.
var IdnTopic = newIdnTopicRegistry()

// ParseTopicDescriptor parses a topic name and constructs a resulting topic descriptor.
func ParseTopicDescriptor(field string) (event.TopicDescriptor, error) {
	r := reflect.ValueOf(IdnTopic)
	td := reflect.Indirect(r).FieldByName(strings.ToUpper(field))

	if td.IsValid() {
		return td.Interface().(event.TopicDescriptor), nil
	}

	return nil, fmt.Errorf("invalid topic name: %s", strings.ToUpper(field))
}

// newIdnTopicRegistry constructs a registry for the mapping between topic name and descriptor.
func newIdnTopicRegistry() *idnTopicRegistry {
	return &idnTopicRegistry {
		ACCESS_PROFILE: event.NewSimpleTopicDescriptor(event.TopicScopePod, "access_profile"),
		ACCESS_REQUEST: event.NewSimpleTopicDescriptor(event.TopicScopePod, "access_request"),
		ACCOUNT_AGGREGATION: event.NewSimpleTopicDescriptor(event.TopicScopePod, "account_aggregation"),
		AGGREGATION_HISTORY: event.NewSimpleTopicDescriptor(event.TopicScopePod, "aggregation_history"),
		AUDIT: event.NewSimpleTopicDescriptor(event.TopicScopePod, "audit"),
		AUTHENTICATION: event.NewSimpleTopicDescriptor(event.TopicScopePod, "authentication"),
		BRANDING: event.NewSimpleTopicDescriptor(event.TopicScopePod, "branding"),
		CAM_EVENTS: event.NewSimpleTopicDescriptor(event.TopicScopePod, "cam_events"),
		CAM_REPORT_REQUEST: event.NewSimpleTopicDescriptor(event.TopicScopePod, "cam_report_request"),
		CC: event.NewSimpleTopicDescriptor(event.TopicScopePod, "cc"),
		CMS: event.NewSimpleTopicDescriptor(event.TopicScopePod, "cms"),
		CMS_8P: event.NewSimpleTopicDescriptor(event.TopicScopeOrg, "cms_8p"),
		ENTITLEMENT: event.NewSimpleTopicDescriptor(event.TopicScopePod, "entitlement"),
		IAI_ADMIN: event.NewSimpleTopicDescriptor(event.TopicScopePod, "iai_admin"),
		IDENTITY: event.NewSimpleTopicDescriptor(event.TopicScopePod, "identity"),
		IDENTITY_EVENT: event.NewSimpleTopicDescriptor(event.TopicScopePod, "identity_event"),
		IDENTITY_PROFILE: event.NewSimpleTopicDescriptor(event.TopicScopePod, "identity_profile"),
		IDENTITY_REQUEST: event.NewSimpleTopicDescriptor(event.TopicScopePod, "identity_request"),
		INTERNAL_TEST: event.NewSimpleTopicDescriptor(event.TopicScopePod, "internal_test"),
		IRIS_DELAYED_EVENT: event.NewSimpleTopicDescriptor(event.TopicScopeGlobal, "iris_delayed_event"),
		MANUAL_WORK_ITEM: event.NewSimpleTopicDescriptor(event.TopicScopePod, "manual_work_item"),
		MATERIALIZER_WORK_QUEUE: event.NewSimpleTopicDescriptor(event.TopicScopePod, "materializer_work_queue"),
		NATIVE_CHANGE_DETECTION: event.NewSimpleTopicDescriptor(event.TopicScopePod, "native_change_detection"),
		NON_EMPLOYEE: event.NewSimpleTopicDescriptor(event.TopicScopePod, "non_employee"),
		NOTIFICATION: event.NewSimpleTopicDescriptor(event.TopicScopePod, "notification"),
		ORG_CONFIG: event.NewSimpleTopicDescriptor(event.TopicScopePod, "org_config"),
		ORG_LIFECYCLE: event.NewSimpleTopicDescriptor(event.TopicScopePod, "org_lifecycle"),
		PASSWORD_SYNC_GROUP: event.NewSimpleTopicDescriptor(event.TopicScopePod, "password_sync_group"),
		POST_APPROVAL: event.NewSimpleTopicDescriptor(event.TopicScopePod, "post_approval"),
		POST_PROVISIONING: event.NewSimpleTopicDescriptor(event.TopicScopePod, "post_provisioning"),
		PROVISIONING: event.NewSimpleTopicDescriptor(event.TopicScopePod, "provisioning"),
		RESOURCE_OBJECT: event.NewSimpleTopicDescriptor(event.TopicScopePod, "resource_object"),
		ROLE: event.NewSimpleTopicDescriptor(event.TopicScopePod, "role"),
		ROLE_MINING: event.NewSimpleTopicDescriptor(event.TopicScopePod, "role_mining"),
		SEARCH: event.NewSimpleTopicDescriptor(event.TopicScopePod, "search"),
		SEARCH_ACTION_POD: event.NewSimpleTopicDescriptor(event.TopicScopePod, "search_action_pod"),
		SOD: event.NewSimpleTopicDescriptor(event.TopicScopePod, "sod"),
		SOURCE: event.NewSimpleTopicDescriptor(event.TopicScopePod, "source"),
		TAGS: event.NewSimpleTopicDescriptor(event.TopicScopePod, "tags"),
		TASK_EXECUTION: event.NewSimpleTopicDescriptor(event.TopicScopePod, "task_execution"),
		TASK_SCHEDULE: event.NewSimpleTopicDescriptor(event.TopicScopePod, "task_schedule"),
		TENANT_USAGE: event.NewSimpleTopicDescriptor(event.TopicScopePod, "tenant_usage"),
		TRANSFORM: event.NewSimpleTopicDescriptor(event.TopicScopePod, "transform"),
		TRIGGER: event.NewSimpleTopicDescriptor(event.TopicScopePod, "trigger"),
		TRIGGER_ACK: event.NewSimpleTopicDescriptor(event.TopicScopePod, "trigger_ack"),
		UPDATED_COMPOSITE_IDENTITY: event.NewSimpleTopicDescriptor(event.TopicScopePod, "updated_composite_identity"),
	}
}

// idnTopicRegistry contains a list of IdentityNow topics.
type idnTopicRegistry struct {
	ACCESS_PROFILE event.TopicDescriptor
	ACCESS_REQUEST event.TopicDescriptor
	ACCOUNT_AGGREGATION event.TopicDescriptor
	AGGREGATION_HISTORY event.TopicDescriptor
	AUDIT event.TopicDescriptor
	AUTHENTICATION event.TopicDescriptor
	BRANDING event.TopicDescriptor
	CAM_EVENTS event.TopicDescriptor
	CAM_REPORT_REQUEST event.TopicDescriptor
	CC event.TopicDescriptor
	CMS event.TopicDescriptor
	CMS_8P event.TopicDescriptor
	ENTITLEMENT event.TopicDescriptor
	IAI_ADMIN event.TopicDescriptor
	IDENTITY event.TopicDescriptor
	IDENTITY_EVENT event.TopicDescriptor
	IDENTITY_PROFILE event.TopicDescriptor
	IDENTITY_REQUEST event.TopicDescriptor
	INTERNAL_TEST event.TopicDescriptor
	IRIS_DELAYED_EVENT event.TopicDescriptor
	MANUAL_WORK_ITEM event.TopicDescriptor
	MATERIALIZER_WORK_QUEUE event.TopicDescriptor
	NATIVE_CHANGE_DETECTION event.TopicDescriptor
	NON_EMPLOYEE event.TopicDescriptor
	NOTIFICATION event.TopicDescriptor
	ORG_CONFIG event.TopicDescriptor
	ORG_LIFECYCLE event.TopicDescriptor
	PASSWORD_SYNC_GROUP event.TopicDescriptor
	POST_APPROVAL event.TopicDescriptor
	POST_PROVISIONING event.TopicDescriptor
	PROVISIONING event.TopicDescriptor
	RESOURCE_OBJECT event.TopicDescriptor
	ROLE event.TopicDescriptor
	ROLE_MINING event.TopicDescriptor
	SEARCH event.TopicDescriptor
	SEARCH_ACTION_POD event.TopicDescriptor
	SOD event.TopicDescriptor
	SOURCE event.TopicDescriptor
	TAGS event.TopicDescriptor
	TASK_EXECUTION event.TopicDescriptor
	TASK_SCHEDULE event.TopicDescriptor
	TENANT_USAGE event.TopicDescriptor
	TRANSFORM event.TopicDescriptor
	TRIGGER event.TopicDescriptor
	TRIGGER_ACK event.TopicDescriptor
	UPDATED_COMPOSITE_IDENTITY event.TopicDescriptor
}
