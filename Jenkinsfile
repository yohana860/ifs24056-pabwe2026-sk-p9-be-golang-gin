// Quality gate Delcom Blog (backend Go + frontend React + embed + binary).
// Credential: simpan OPENAI_KEY di Jenkins Credentials (Secret text, id: openai-key).
pipeline {
  agent any
  options { timestamps(); timeout(time: 30, unit: 'MINUTES') }
  parameters {
    string(name: 'FE_REPO_URL', defaultValue: 'https://github.com/auxtern/REPO-FRONTEND.git', description: 'URL repo frontend React pasangan')
    string(name: 'FE_BRANCH', defaultValue: 'main', description: 'Branch frontend')
  }
  environment {
    GOFLAGS = '-mod=mod'
    APP_ENV = 'test'
    APP_PORT = '8081'
    DB_HOST = '127.0.0.1'
    DB_PORT = '5432'
    DB_DATABASE = 'pabwe_p9_blog'
    DB_USERNAME = 'postgres'
    DB_PASSWORD = 'postgres'
    APP_KEY = 'jenkins_secret_key_super_secure_2026_random_string_32chars'
  }
  stages {
    stage('Checkout') {
      steps {
        checkout scm
        dir('frontend') { git url: params.FE_REPO_URL, branch: params.FE_BRANCH }
      }
    }
    stage('Setup Tools') {
      steps { sh 'go version && node --version && npm --version' }
    }
    stage('Setup PostgreSQL 17') {
      steps { sh 'docker rm -f pg17-ci || true; docker run -d --name pg17-ci -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=pabwe_p9_blog -p 5432:5432 postgres:17; for i in $(seq 1 30); do docker exec pg17-ci pg_isready -U postgres && break; sleep 1; done' }
    }
    stage('Backend: dependencies & format') {
      steps {
        sh 'go mod download'
        sh 'test -z "$(gofmt -l .)" || (gofmt -l . && exit 1)'
        sh 'go vet ./...'
      }
    }
    stage('Backend: tests & 100% coverage') {
      steps {
        sh 'go test ./... -coverprofile=coverage.out'
        sh 'go tool cover -func=coverage.out | tee coverage.txt'
        sh './scripts/check-coverage.sh coverage.out'
      }
    }
    stage('Frontend: lint, tests & 100% coverage') {
      steps {
        dir('frontend') {
          sh 'npm ci'
          sh 'npm run lint'
          sh 'npm run test -- --coverage'   // threshold 100% di vite.config.js => gagal bila kurang
        }
      }
    }
    stage('Build: frontend -> embed -> binary') {
      steps { sh 'FE_DIR=frontend ./scripts/build.sh' }
    }
    stage('Binary smoke test') {
      steps {
        withCredentials([string(credentialsId: 'openai-key', variable: 'OPENAI_KEY')]) {
          sh './scripts/smoke.sh'
        }
      }
    }
  }
  post {
    always {
      sh 'docker rm -f pg17-ci || true'
      archiveArtifacts artifacts: 'coverage.out, coverage.txt, bin/server, frontend/coverage/**', allowEmptyArchive: true
    }
  }
}
