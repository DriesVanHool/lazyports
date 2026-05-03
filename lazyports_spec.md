# LazyPorts — Technical & Functional Specification

## Overview

LazyPorts is a fast, cross-platform terminal UI (TUI) tool for viewing and managing processes bound to network ports.

It replaces workflows like:
- lsof -i :PORT
- kill -9 PID

with a fast, interactive interface.

---

## Goals

- Instantly see which processes use which ports
- Quickly kill processes
- Clean, fast TUI
- Cross-platform (macOS, Linux, Windows)

---

## Core Features

### Port Table
Columns:
- Port
- Process
- PID
- Protocol
- State

### Search
- `/` to search
- Fuzzy match

### Actions
- k: kill
- r: refresh
- Enter: details
- q: quit

### CLI Mode
lazyports list
lazyports kill 8080

---

## Tech Stack

- Go
- Bubble Tea
- Lip Gloss

---

## Installation (README)

### macOS/Linux
curl -sSL https://example.com/install.sh | bash

### Windows
Download binary and run:
lazyports.exe

---

## Usage

lazyports

---

## Why

Because killing ports manually is annoying.
