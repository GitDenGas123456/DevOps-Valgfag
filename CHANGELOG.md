# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-12-02

### Added

- Initial release of the WhoKnows web application
- User authentication (login, register, logout)
- Search functionality with optional FTS (Full-Text Search) support
- Weather page integration
- RESTful API endpoints for authentication and search
- Swagger API documentation
- Health check endpoint (`/healthz`)
- Prometheus metrics endpoint (`/metrics`)
- Docker support with docker-compose configuration
- SQLite database with migration support
- CI/CD pipeline with GitHub Actions
- Monitoring setup with Prometheus and Grafana

### Infrastructure

- Dockerfile for containerized deployment
- Docker Compose configurations for development and monitoring
- Automated testing and linting in CI pipeline
- Automated deployment to production environment
