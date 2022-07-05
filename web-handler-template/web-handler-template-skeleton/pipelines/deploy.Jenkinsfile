// Using jenkins-release-utils https://github.com/sailpoint/jenkins-release-utils
@Library('sailpoint/jenkins-release-utils')_

currentBuild.description = """Service: sp-connect\n Build: ${params.buildNumber}\n Pod: ${params.podName}\n Branch: ${params.branch}"""

def getModuleName() {
    return env.POD_NAME != 'megapod-useast1' ? env.SERVICE : env.SERVICE
}

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
        BUILD_NUMBER = "${params.buildNumber}"
        POD_NAME = "${params.podName}".toLowerCase()
        BRANCH = "${params.branch}"
        ECR_REPOSITORY = '406205545357.dkr.ecr.us-east-1.amazonaws.com'
        ECR_REPOSITORY_NAME = "sailpoint/${env.SERVICE}"

        // Which channel to report any stage updates to
        SLACK_CHANNEL = "#team-eng-platform-connectivity-jnk"
    }

    stages {
        stage('Deploy') {
            steps {
                echo "${env.JOB_NAME} - ${env.BUILD_DISPLAY_NAME} Deploy Started"

                container('release-utils') {
                    sh "drydock_deploy.sh ${env.POD_NAME} ${getModuleName()} ${env.BUILD_NUMBER}"
                }

                echo "${env.JOB_NAME} - ${env.BUILD_DISPLAY_NAME} Deploy Finished"
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