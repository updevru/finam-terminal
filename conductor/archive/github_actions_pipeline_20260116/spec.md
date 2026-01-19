# Specification: GitHub Actions CI/CD Pipeline & Documentation

## Overview
This track aims to automate the testing, linting, building, and distribution of the `finam-terminal` application. We will implement a GitHub Actions workflow to ensure code quality on Pull Requests and automate multi-platform binary releases and Docker image publication when a new release is created. Additionally, we will update the documentation to provide clear installation instructions for all supported platforms.

## Functional Requirements
- **Pull Request Automation:**
    - Run unit tests on every Pull Request to the main branch.
    - Execute `golangci-lint` to ensure code style and catch potential errors.
- **Release Automation:**
    - Triggered upon the creation of a GitHub Release.
    - Build binaries for Windows (amd64), Linux (amd64), and macOS (amd64, arm64).
    - Automatically attach these binaries to the GitHub Release.
- **Docker Integration:**
    - Build a Docker image of the application.
    - Publish the image to GitHub Container Registry (ghcr.io) when a release is created.
- **Documentation:**
    - Update `README.md` with specific installation and setup instructions for Windows, Linux, macOS, and Docker.

## Non-Functional Requirements
- **Security:** Use GitHub Secrets for any sensitive information (though the registry is public, the actor is the workflow).
- **Efficiency:** Optimize the Docker build process and workflow execution time.

## Acceptance Criteria
- [ ] PRs cannot be merged without passing tests and linting.
- [ ] Creating a release automatically generates and attaches 4 binaries (Windows, Linux, macOS x2).
- [ ] A Docker image is successfully pushed to `ghcr.io` upon release.
- [ ] `README.md` contains a clear "Installation" section covering all OS options.

## Out of Scope
- Integration tests requiring real API tokens (unit tests only).
- Automatic semantic versioning (tagging will be done manually by the user).
