@Library("sailpoint/jenkins-release-utils")_

pipeline {
    environment {
        SLACK_CHANNEL = "#team-eng-platform-connectivity-jnk"
        SERVICE_NAME = "sp-connect"
        GIT_REPO_NAME = "saas-sp-connect"
    }

    agent {
        kubernetes {
            yaml """\
                apiVersion: v1
                kind: Pod
                spec:
                  serviceAccount: pod-build-container
                  securityContext:
                    fsGroup: 65534
                  containers:
                  - name: go
                    image: golang:1.17.1-alpine3.14
                    imagePullPolicy: IfNotPresent
                    command:
                    - cat
                    tty: true
                  - name: aws-cli
                    image: 406205545357.dkr.ecr.us-east-1.amazonaws.com/devops/aws-cli-jq:2132
                    imagePullPolicy: IfNotPresent
                    command:
                    - cat
                    tty: true
                  tolerations:
                  - key: cb-core
                    operator: Equal
                    value: agent
                    effect: NoSchedule
                """.stripIndent()
            }
    }

    stages {
        stage('Checkout SCM') {
            steps {
                checkout(
                [$class: "GitSCM",
                branches: [[name: "origin/${branch}"]],
                doGenerateSubmoduleConfigurations: false,
                extensions: [], submoduleCfg: [],
                userRemoteConfigs: [[credentialsId: "git-automation-ssh", url: "git@github.com:sailpoint/${env.GIT_REPO_NAME}.git"]]])
            }
        }
        stage('Run E2E tests') {
            steps {
                container('aws-cli') {
                    assumePodRole {
                        container('go') {
                            script {
                                sh """
                                    apk add --no-cache iproute2 build-base
                                    go test -v ./api-test/... \
                                    -username ${username} -password ${password} -url ${apiUrl} -env dev
                                """
                                
                            } 
                        }
                    }
                }
            }
        }
    }
    
    post {
        success {
            sendSlackNotification(
                env.SLACK_CHANNEL,
                "${env.SERVICE_NAME} E2E verify pipeline for <${env.BUILD_URL}|${env.BUILD_NUMBER}> was successful.",
                utils.NOTIFY_SUCCESS
            )
        }
        failure {
            sendSlackNotification(
                env.SLACK_CHANNEL,
                "${env.SERVICE_NAME} E2E verify pipeline for <${env.BUILD_URL}|${env.BUILD_NUMBER}> failed.",
                utils.NOTIFY_FAILURE
            )
        }
        aborted {
            sendSlackNotification(
                env.SLACK_CHANNEL,
                "${env.SERVICE_NAME} E2E verify pipeline for <${env.BUILD_URL}|${env.BUILD_NUMBER}> was aborted.",
                utils.NOTIFY_ABORTED
            )
        }
    }
}
