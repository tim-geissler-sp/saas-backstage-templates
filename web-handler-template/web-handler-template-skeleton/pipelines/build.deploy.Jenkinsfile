// Using jenkins-release-utils https://github.com/sailpoint/jenkins-release-utils
@Library('sailpoint/jenkins-release-utils')_

currentBuild.description = """Service: sp-connect\n Branch: ${params.branch}\n Pod: ${params.podName}
Version: ${env.BUILD_NUMBER}"""

def getModuleName() {
    return env.POD_NAME != 'megapod-useast1' ? env.SERVICE : env.SERVICE
}

def buildJob = ""

pipeline {
    agent {
        kubernetes {
            yaml "${libraryResource 'pods/build-container.yaml'}"
        }
    }

    // You can find a full list of options here: https://www.jenkins.io/doc/book/pipeline/syntax/#options
    options {
        // Add timestamps to console output
        timestamps()

        // Aborts job if run time is over a day
        timeout(time: 3, unit: "HOURS")

        // Keep the last 20 builds (For Auditing)
        buildDiscarder(logRotator(numToKeepStr: "20"))

        // Don't allow concurrent builds to run
        disableConcurrentBuilds()
    }

    environment {
        SERVICE = "sp-connect"
        GIT_SERVICE_NAME = "saas-sp-connect"
        BRANCH = "${params.branch}"
        POD_NAME = "${params.podName}".toLowerCase()
        ECR_REPOSITORY = '406205545357.dkr.ecr.us-east-1.amazonaws.com'
        ECR_REPOSITORY_NAME = "sailpoint/${env.SERVICE}"
        VERSION = "${env.BUILD_NUMBER}"


        // Which channel to report any stage updates to
        SLACK_CHANNEL = "#team-eng-platform-connectivity-jnk"
    }

    stages {
        stage('Container Build & Push') {
            steps {
                echo "${env.JOB_NAME} - ${env.BUILD_DISPLAY_NAME} Started"

                slackSend(
                    channel: env.SLACK_CHANNEL,
                    message: "${env.JOB_NAME} - ${env.BUILD_DISPLAY_NAME} Started (<${env.BUILD_URL}|Open>)",
                    color: 'good'
                )
                script {
                    buildJob = build job: 'build-sp-connect', parameters: [
                            [$class: 'StringParameterValue', name: 'branch', value: "${env.BRANCH}"]
                        ]
                    }
            }
        }

        stage('Deploy') {
            steps {
                build job: 'deploy-sp-connect', parameters: [
                			[$class: 'StringParameterValue', name: 'buildNumber', value: "${buildJob.getNumber()}"],
                			[$class: 'StringParameterValue', name: 'podName', value: "${env.POD_NAME}"],
                            [$class: 'StringParameterValue', name: 'branch', value: "${env.BRANCH}"]
                		], wait: true
            }
        }
    }

    post {
            success {
                slackSend(
                    channel: env.SLACK_CHANNEL,
                    message: "${env.JOB_NAME} - ${env.BUILD_DISPLAY_NAME} pipeline was successful (<${env.BUILD_URL}|Open>)",
                    color: 'good'
                )
            }
            failure {
                slackSend(
                    channel: env.SLACK_CHANNEL,
                    message: "${env.JOB_NAME} - ${env.BUILD_DISPLAY_NAME} pipeline failed (<${env.BUILD_URL}|Open>)",
                    color: 'danger'
                )
            }
            aborted {
                slackSend(
                    channel: env.SLACK_CHANNEL,
                    message: "${env.JOB_NAME} - ${env.BUILD_DISPLAY_NAME} pipeline was aborted (<${env.BUILD_URL}|Open>)",
                    color: 'danger'
                )
            }
    }
}