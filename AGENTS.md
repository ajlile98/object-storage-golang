# Repository Guidance

## Purpose

This is a personal learning project for building an object storage service in Go. Prioritize clear explanations that build understanding over completing work quickly.

## Collaboration

- Keep responses short, direct, and focused on the question asked.
- Do not modify files, run commands that change state, or add dependencies unless the user explicitly asks for an implementation or execution.
- When explaining code, describe the relevant Go concepts and the reason for the design choice.
- Make assumptions explicit when they affect an answer.

## Engineering

- Follow idiomatic Go: small focused packages, explicit error handling, `gofmt`, and standard-library solutions where practical.
- Apply SOLID and DRY pragmatically. Do not introduce interfaces, layers, or abstractions without a concrete current need.
- Prefer composition and dependency injection over inheritance-style designs.
- Keep HTTP concerns, object-service logic, and blob-storage implementations separate.
- Treat input validation and storage-path safety as defense-in-depth concerns.
- Follow Twelve-Factor principles when applicable, especially configuration through the environment and stateless service behavior. Do not force them into a local learning project where they add no value.
- Keep changes small and scoped; avoid unrelated refactors.

## Validation

- For requested code changes, run the narrowest relevant formatter, test, or build check when available.
- Report validation results and any limitations concisely.