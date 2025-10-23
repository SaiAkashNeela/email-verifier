# AI Rules for Email Validator Service

This document outlines the technical stack and guidelines for library usage within the Email Validator Service. These rules are designed to maintain consistency, performance, and readability across the codebase.

## Tech Stack

The Email Validator Service is built using a modern, efficient, and observable Go-based architecture.

*   **Go (1.21+)**: The primary programming language, chosen for its performance, concurrency features, and robust standard library.
*   **HTTP Server**: Utilizes Go's `net/http` package or a lightweight, idiomatic router for building RESTful API endpoints.
*   **Redis**: Employed as an in-memory data store for caching domain validation results, enhancing performance and reducing external lookups.
*   **Prometheus**: Integrated for collecting and exposing application metrics, providing real-time insights into service health and performance.
*   **Grafana**: Used in conjunction with Prometheus to visualize metrics through dashboards, enabling effective monitoring and alerting.
*   **Docker & Docker Compose**: Essential for containerizing the application and its dependencies, ensuring consistent development, testing, and deployment environments.
*   **GitHub Actions**: Implemented for Continuous Integration and Continuous Deployment (CI/CD), automating testing, linting, and deployment workflows.
*   **OpenAPI/Swagger**: Used to define and document the API, ensuring clear communication of endpoints, request/response schemas, and overall API contract.
*   **`golangci-lint`**: A fast Go linters aggregator, used to enforce code quality, style, and identify potential issues early in the development cycle.

## Library Usage Rules

To maintain a clean, efficient, and maintainable codebase, adhere to the following library usage rules:

*   **Core Logic**: Always prioritize the Go standard library for fundamental programming tasks, data structures, concurrency primitives, and basic HTTP handling. Avoid external dependencies where a standard library solution is sufficient.
*   **HTTP Routing**: For defining API routes and handling HTTP requests, use Go's `net/http` package. If a more advanced routing solution is required, a lightweight and widely adopted Go HTTP router may be considered, but must be approved.
*   **Caching**: All caching mechanisms must interact with Redis using a dedicated Go Redis client library (e.g., `go-redis`). Ensure cache keys are well-defined and expiration policies are appropriate.
*   **Metrics**: For exposing application metrics to Prometheus, use the official Prometheus Go client library. Define clear, descriptive metric names and labels.
*   **Configuration**: Configuration should primarily be managed through environment variables or a simple, standard Go configuration parsing library (e.g., `flag` or `viper` if introduced).
*   **Testing**: Utilize Go's built-in `testing` package for all unit and integration tests. For mocking interfaces in tests, `mockgen` should be used to generate mock implementations.
*   **Linting**: All Go code must pass `golangci-lint` checks with the configured rules. Integrate linting into your local development workflow and CI/CD pipeline.
*   **Dependency Management**: Go Modules (`go mod`) is the sole tool for managing project dependencies. Ensure `go mod tidy` is run regularly to keep `go.mod` and `go.sum` files clean and up-to-date.
*   **Error Handling**: Follow Go's idiomatic error handling practices, returning errors explicitly rather than relying on exceptions. Avoid `panic` for recoverable errors.