// Using jenkins-release-utils https://github.com/sailpoint/jenkins-release-utils
@Library('sailpoint/jenkins-release-utils')_

currentBuild.description = """Service: sp-connect\n Branch: ${params.branch}\n Version: ${env.BUILD_NUMBER}"""

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
        ECR_REPOSITORY = '406205545357.dkr.ecr.us-east-1.amazonaws.com'
        ECR_REPOSITORY_NAME = "sailpoint/${env.SERVICE}"
        VERSION = "${env.BUILD_NUMBER}"

        // Which channel to report any stage updates to
        SLACK_CHANNEL = "#team-eng-platform-connectivity-jnk"
    }

    stages {

        stage('Checkout SCM') {
            steps {
                echo "${env.JOB_NAME} - ${env.BUILD_DISPLAY_NAME} Checkout SCM Started"

                slackSend(
                    channel: env.SLACK_CHANNEL,
                    message: "${env.JOB_NAME} - ${env.BUILD_DISPLAY_NAME} Started (<${env.BUILD_URL}|Open>)",
                    color: 'good'
                )

                checkout(
                [$class: 'GitSCM',
                branches: [[name: "origin/${env.BRANCH}"]],
                doGenerateSubmoduleConfigurations: false,
                extensions: [], submoduleCfg: [],
                userRemoteConfigs: [[credentialsId: 'git-automation-ssh', url: "git@github.com:sailpoint/${env.GIT_SERVICE_NAME}.git"]]])

                echo "${env.JOB_NAME} - ${env.BUILD_DISPLAY_NAME} Checkout SCM Finished"
            }
        }

        stage('Container Build & Push') {
            steps {
                echo "${env.JOB_NAME} - ${env.BUILD_DISPLAY_NAME} Container Build & Push Started"

                container('kaniko') {
                    script {
                        sh """
                            /kaniko/executor \
                            --context . \
                            --dockerfile Dockerfile \
                            --build-arg version=${env.VERSION} \
                            --destination=${env.ECR_REPOSITORY}/${env.ECR_REPOSITORY_NAME}:${env.VERSION}
                        """
                    }
                }

                echo "${env.JOB_NAME} - ${env.BUILD_DISPLAY_NAME} Container Build & Push Finished"
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