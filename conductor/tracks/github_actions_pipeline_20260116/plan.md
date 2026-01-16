# Implementation Plan: GitHub Actions CI/CD Pipeline & Documentation

This plan outlines the steps to automate testing, linting, building, and distribution of the `finam-terminal` application, along with updating documentation for all supported platforms.

## Phase 1: CI Pipeline (Tests & Linting)
- [ ] Task: Configure `golangci-lint` locally
    - [ ] Create or update `.golangci.yml` with project-specific linting rules.
    - [ ] Run `golangci-lint run` locally and fix any existing linting issues to ensure a clean baseline.
- [ ] Task: Create PR workflow file (`.github/workflows/ci.yml`)
    - [ ] Define a workflow triggered on pull requests to the main branch.
    - [ ] Add a job to setup the Go environment.
    - [ ] Add steps to run `go test ./...` and `golangci-lint`.
- [ ] Task: Verify PR workflow
    - [ ] Push changes to a branch and verify the workflow triggers and passes in GitHub.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: CI Pipeline' (Protocol in workflow.md)

## Phase 2: Build & Release Automation
- [ ] Task: Create Release workflow file (`.github/workflows/release.yml`)
    - [ ] Define a workflow triggered on tag creation (e.g., `v*`).
- [ ] Task: Add multi-platform build steps
    - [ ] Configure a build matrix or separate steps for Windows (amd64), Linux (amd64), and macOS (amd64, arm64).
    - [ ] Use `GOOS` and `GOARCH` environment variables for cross-compilation.
- [ ] Task: Add asset upload logic
    - [ ] Use a GitHub Action (like `softprops/action-gh-release`) to upload compiled binaries to the release.
- [ ] Task: Verify Release workflow
    - [ ] Create a dummy tag and verify that binaries are correctly built and attached to the release.
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Build & Release Automation' (Protocol in workflow.md)

## Phase 3: Dockerization
- [ ] Task: Create `Dockerfile`
    - [ ] Create a multi-stage `Dockerfile` to build the Go application and create a minimal runtime image.
    - [ ] Verify the Docker image builds and runs correctly locally.
- [ ] Task: Update Release workflow for Docker
    - [ ] Add steps to log in to GitHub Container Registry (ghcr.io).
    - [ ] Add steps to build and push the Docker image when a release is created.
- [ ] Task: Verify Docker image in GHCR
    - [ ] Pull the image from `ghcr.io` and verify it runs as expected.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Dockerization' (Protocol in workflow.md)

## Phase 4: Documentation
- [ ] Task: Update `README.md` with Installation instructions
    - [ ] Add a "Installation" section.
    - [ ] Provide clear, step-by-step instructions for Windows, Linux, and macOS (downloading binaries from releases).
    - [ ] Add a subsection for Docker usage (pulling and running the image).
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Documentation' (Protocol in workflow.md)
