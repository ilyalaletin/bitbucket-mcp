# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Bitbucket MCP server — a Go project implementing a Model Context Protocol (MCP) server for Bitbucket integration. Licensed under MIT.

## Language & Build

- **Language**: Go
- **Build**: `go build ./...`
- **Test**: `go test ./...` (single test: `go test -run TestName ./path/to/package`)
- **Lint**: `go vet ./...`

## Development Rules

- **Branching:** All changes go in feature branches, never commit directly to main.
- **TDD:** Write tests first, see them fail, then implement. No code without a failing test.
- **Green tests required:** All tests must pass before merging. Run `go test ./...` to verify.

## Project Status

This is a new project. As the codebase grows, update this file with architecture details, key packages, and development conventions.
