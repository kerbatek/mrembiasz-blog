def githubCheck(String name, Closure body) {
  publishChecks(
    name: name,
    status: 'IN_PROGRESS',
    summary: 'Running',
    detailsURL: env.BUILD_URL
  )

  try {
    def checkResult = body.call()
    def checkTitle = null
    def checkSummary = 'Passed'
    def checkText = null

    if (checkResult instanceof Map) {
      checkTitle = checkResult.title ?: checkTitle
      checkSummary = checkResult.summary ?: checkSummary
      checkText = checkResult.text
    } else if (checkResult) {
      checkSummary = checkResult
    }

    def checkArgs = [
      name: name,
      status: 'COMPLETED',
      conclusion: 'SUCCESS',
      summary: checkSummary,
      detailsURL: env.BUILD_URL
    ]

    if (checkTitle) {
      checkArgs.title = checkTitle
    }

    if (checkText) {
      checkArgs.text = checkText
    }

    publishChecks checkArgs
  } catch (error) {
    publishChecks(
      name: name,
      status: 'COMPLETED',
      conclusion: 'FAILURE',
      summary: 'Failed',
      text: error.getMessage(),
      detailsURL: env.BUILD_URL
    )
    throw error
  }
}

def checkedSh(Map args) {
  githubCheck(args.checkName ?: args.name) {
    sh args.exec
    def result = [
      summary: args.name
    ]
    if (args.titleFile) {
      result.title = readFile(args.titleFile).trim()
    }
    if (args.textFile) {
      result.text = readFile(args.textFile).trim()
    }
    return result
  }
}

def withAwsRole(String containerName, String roleArn, Closure body) {
  container(containerName) {
    withEnv(["AWS_ROLE_ARN=${roleArn}", 'AWS_DEFAULT_REGION=eu-central-1']) {
      withCredentials([file(credentialsId: 'aws-oidc-token', variable: 'AWS_WEB_IDENTITY_TOKEN_FILE')]) {
        body.call()
      }
    }
  }
}

def checkedAwsSh(Map args) {
  withAwsRole(args.container, args.role) {
    checkedSh(name: args.name, checkName: args.checkName, exec: args.exec, titleFile: args.titleFile)
  }
}

def checkedSonarScan(Map args) {
  container('sonarqube') {
    withSonarQubeEnv('SonarQube') {
      checkedSh(
        name: args.name,
        checkName: args.checkName,
        exec: "deploy/scripts/sonarqube-scan.sh ${args.component}",
        textFile: "sonar-${args.component}-dashboard-url.txt"
      )
    }
  }
}

def checkedQualityGate(Map args) {
  def dashboardUrlFile = "sonar-${args.component}-dashboard-url.txt"
  if (!fileExists(dashboardUrlFile)) {
    echo "${args.name}: skipped because scan failed."
    return
  }

  githubCheck(args.checkName) {
    def qualityGate = timeout(time: 15, unit: 'MINUTES') {
      waitForQualityGate abortPipeline: true
    }
    return [
      summary: "${args.name}: ${qualityGate.status}",
      text: readFile(dashboardUrlFile).trim()
    ]
  }
}

return this
