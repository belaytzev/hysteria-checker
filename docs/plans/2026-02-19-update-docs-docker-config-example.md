# Update Docs, Docker Compose, and Config Example File

## Overview
Update docker-compose.yml to include all configuration options as commented-out environment variables, update the README docker-compose example to match, and create a config.example.env file showing all available options with descriptions.

## Context
- Files involved: `docker-compose.yml`, `README.md`, `config.example.env` (new)
- Related patterns: Config options defined in `config/config.go` using kong tags
- Dependencies: None

## Development Approach
- **Testing approach**: Regular (no automated tests needed - docs/config only)
- Complete each task fully before moving to the next
- All changes must stay in sync with `config/config.go` as the source of truth

## Implementation Steps

### Task 1: Create config.example.env

**Files:**
- Create: `config.example.env`

- [x] Create `config.example.env` with all 13 config options from `config/config.go`
- [x] Each option should have a comment with its description and default value
- [x] Required options (SUBSCRIPTION_URL) uncommented with placeholder
- [x] Optional options commented out with their defaults shown
- [x] Group options logically: subscription, check settings, metrics/web server, auth, logging

### Task 2: Update docker-compose.yml with all config options

**Files:**
- Modify: `docker-compose.yml`

- [ ] Add all missing environment variables as commented-out entries with defaults
- [ ] Keep SUBSCRIPTION_URL, CHECK_INTERVAL, CHECK_METHOD, LOG_LEVEL uncommented (current state)
- [ ] Add commented-out entries for: CHECK_URL, CHECK_TIMEOUT, METRICS_HOST, METRICS_PORT, METRICS_PROTECTED, METRICS_USERNAME, METRICS_PASSWORD, WEB_PUBLIC
- [ ] Add inline comments showing default values for commented-out options

### Task 3: Update README.md docker-compose example

**Files:**
- Modify: `README.md`

- [ ] Update the Quick Start docker-compose.yml snippet to include all environment variables (commented-out with defaults, matching docker-compose.yml)
- [ ] Ensure the example stays concise but comprehensive
- [ ] Add a note pointing users to `config.example.env` for reference

### Task 4: Verify consistency

- [ ] Cross-check all 13 config options appear in: `config.example.env`, `docker-compose.yml`, `README.md` config table
- [ ] Verify defaults match between all files and `config/config.go`
- [ ] Run `git diff` to review all changes for correctness
