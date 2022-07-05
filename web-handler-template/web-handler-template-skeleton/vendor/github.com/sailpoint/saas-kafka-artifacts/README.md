# Saas kafka artifacts

- [**Summary**](#summary)
- [**Installation and upgrade**](#installation-and-upgrade)
  - [Prerequisite](#prerequisite)
  - [Versioning](#versioning)
  - [Java and groovy based services](#java-and-groovy-based-services)
  - [GO based services](#go-based-services)
- [**Create a new topic**](#create-a-new-topic)
  - [Format and specification](#format-and-specification)
  - [Pull request](#pull-request)
  - [Create the new topic in kafka](#create-the-new-topic-in-kafka)

# Summary

This repository defines a new way to define, distribute and consume kafka topics that are used in IdentityNow. Topics will be defined in JSON files. Once the PR is approved and merged, artifacts will be automatically generated and pushed to different locations to be consumed by Java, Groovy, Golang and more.

# Installation and upgrade

## Prerequisite
For atlas, mantis-platform and atlas-boot based services, make sure to upgrade them to the latest before using this new implementation of kafka topics. Failure to do so will result duplicated class error or multiple IdnTopic class.

## Versioning
The artifacts are built from [this Jenkins job on construct](https://construct.identitysoon.com/view/Echo/job/saas-kafka-artifacts/). You will be able to find the latest library from the build number of the Jenkins job. It will be referenced as `VERSION` below.

## Java and groovy based services
Import the kafka topics library with gradle:
```
implementation('com.sailpoint:saas-kafka-topics:VERSION')
```
After that, you should be able to get `TopicDescriptor` from `IdnTopic` Enum provided from this library and start sending events.

## GO based services
Artifact for Golang is hosted in this repository as tags. Since go module enforces semantic versioning, the version of the kafka topics module will be `v1.0.VERSION`.

Add the kafka topic GO module:
```
require (
	github.com/sailpoint/saas-kafka-artifacts v1.0.VERSION
)
```
A package called `topics` will be available to your service. Finally, you will be able to reference to `TopicDescriptor by` `topics.IdnTopic` struct.

# Create a new topic

## Format and specification
Topics are defined in JSON files located in the `topics` folder. Each topic has its own JSON file. These are the required fields within the file:
- `name`: The name of the topic. It should be a string in both lowercase and snake_case.
- `partitionCount`: The number of partitions this topic should have. This should be a positive integer.
- `topicScope`: The scope of the topic. It's a all-cap string and it could be `POD`, `ORG` or `GLOBAL`. 

After the build process, topic will be created in the Enum in all cap and snake_case format. Here's a sample JSON file for identity topic (`IDENTITY.json`).
```
{
    "name": "identity",
    "partitionCount": 16,
    "topicScope": "POD"
}
```

## Pull request
Topic creation requires CAB review. Make sure you attended CAB review and got approval before creating a PR to this repo.

## Create the new topic in kafka
Topics are managed by OLS microservice. After merging the PR, bump up the library version in OLS so that it knows the new topic. Finally, request devops to run upgrader so that OLS actually creates the topic in kafka.
