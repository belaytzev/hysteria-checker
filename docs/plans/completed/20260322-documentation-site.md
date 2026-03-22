# Documentation Site Implementation

## Overview
- Implement an MkDocs Material documentation site per the design spec at `docs/superpowers/specs/2026-03-22-documentation-site-design.md`
- All docs site files live in `site/` at the project root
- 7 documentation pages covering the full hysteria-checker feature set
- Abstract geometric SVG logo and favicon
- Cloudflare Pages deployment via Wrangler

## Context (from discovery)
- Design spec: `docs/superpowers/specs/2026-03-22-documentation-site-design.md`
- Source code to reference: `config/config.go`, `web/api.go`, `web/router.go`, `metrics/metrics.go`, `parser/hysteria1.go`, `parser/hysteria2.go`, `checker/checker.go`, `models/proxy.go`
- Existing dashboard color palette in `web/templates/index.html` (must match)
- Docker setup: `Dockerfile`, `docker-compose.yml`, `config.example.env`

## Development Approach
- **testing approach**: Build verification — run `mkdocs build` after each content task to verify site builds successfully
- Complete each task fully before moving to the next
- Content must be accurate to current codebase (read source files to verify details)
- **CRITICAL: verify `mkdocs build` succeeds before starting next task**
- **CRITICAL: update this plan file when scope changes during implementation**

## Testing Strategy
- **build tests**: `cd site && .venv/bin/mkdocs build --strict` after each task (strict mode catches broken links/references)
- **serve test**: `cd site && mkdocs serve` for visual verification at milestones
- No unit tests needed (documentation project, not code)

## Progress Tracking
- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with ➕ prefix
- Document issues/blockers with ⚠️ prefix

## Implementation Steps

### Task 1: Scaffold MkDocs project structure

**Files:**
- Create: `site/mkdocs.yml`
- Create: `site/requirements.txt`
- Create: `site/docs/index.md` (placeholder)

- [x] Create `site/` directory structure: `site/docs/assets/`
- [x] Create `site/requirements.txt` with mkdocs and mkdocs-material dependencies
- [x] Create `site/mkdocs.yml` with full Material theme configuration per design spec (dark/light toggle, color palette, navigation, search, code highlighting)
- [x] Create placeholder `site/docs/index.md` with a title
- [x] Create Python venv inside `site/`: `cd site && uv venv && uv pip install -r requirements.txt`
- [x] Add `site/.venv/` and `site/site/` to `.gitignore`
- [x] Verify: `cd site && .venv/bin/mkdocs build --strict` succeeds

### Task 2: Create logo and favicon

**Files:**
- Create: `site/docs/assets/logo.svg`
- Create: `site/docs/assets/favicon.png`

- [x] Create `site/docs/assets/logo.svg` — abstract geometric circle with checkmark/pulse motif, primary `#7c4dff`, secondary `#00c853`, legible at 32x32
- [x] Generate `site/docs/assets/favicon.png` (32x32 raster from SVG using `cairosvg` or `rsvg-convert`; add conversion tool to requirements if needed)
- [x] Reference logo and favicon in `site/mkdocs.yml`
- [x] Verify: `cd site && .venv/bin/mkdocs build --strict` succeeds

### Task 3: Write index.md — Home page

**Files:**
- Modify: `site/docs/index.md`

- [x] Brief explanation of Hysteria protocol (2-3 sentences for newcomers)
- [x] What hysteria-checker does and why you'd use it
- [x] Key capabilities bullet list (sourced from actual codebase features)
- [x] Links to Getting Started page
- [x] Verify: `cd site && .venv/bin/mkdocs build --strict` succeeds

### Task 4: Write getting-started.md

**Files:**
- Create: `site/docs/getting-started.md`

- [x] Prerequisites section (Docker or Go 1.25+)
- [x] Docker Compose setup (step-by-step with yaml example from `docker-compose.yml`)
- [x] Docker standalone (`docker build` + `docker run`)
- [x] From source (`go build` + run)
- [x] Verification steps (dashboard, metrics, API status endpoint)
- [x] Common issues / troubleshooting section
- [x] Verify: `cd site && .venv/bin/mkdocs build --strict` succeeds

### Task 5: Write configuration.md

**Files:**
- Create: `site/docs/configuration.md`

- [x] Read `config/config.go` to verify all current options and defaults
- [x] Group options by category: Subscription, Check, Server/Metrics, Authentication, Logging
- [x] Document each option: env var, CLI flag, type, default, description with examples
- [x] Add complete configuration examples for common scenarios
- [x] Verify: `cd site && .venv/bin/mkdocs build --strict` succeeds

### Task 6: Write api-reference.md

**Files:**
- Create: `site/docs/api-reference.md`

- [x] Read `web/router.go` and `web/api.go` to verify endpoints and response structures
- [x] Document each endpoint: method, path, description, curl example, response JSON, status codes
- [x] Include authentication section with full auth matrix: dashboard protected when `METRICS_PROTECTED=true` AND `WEB_PUBLIC=false`; API endpoints protected when `METRICS_PROTECTED=true`; document the conditional behavior
- [x] Include 404 and error response examples
- [x] Verify: `cd site && .venv/bin/mkdocs build --strict` succeeds

### Task 7: Write metrics.md

**Files:**
- Create: `site/docs/metrics.md`

- [x] Read `metrics/metrics.go` to verify metric names and labels
- [x] Available metrics table with label descriptions
- [x] PromQL query examples (down proxy count, average latency, alert conditions)
- [x] Grafana integration tips
- [x] Sample Prometheus alertmanager rules
- [x] Verify: `cd site && .venv/bin/mkdocs build --strict` succeeds

### Task 8: Write uri-formats.md

**Files:**
- Create: `site/docs/uri-formats.md`

- [x] Read `parser/hysteria1.go` and `parser/hysteria2.go` to verify format details
- [x] Hysteria v1 URI format — annotated parameter breakdown
- [x] Hysteria v2 URI format — annotated breakdown, `hy2://` alias
- [x] Port hopping syntax and examples
- [x] Edge cases and validation rules
- [x] Verify: `cd site && .venv/bin/mkdocs build --strict` succeeds

### Task 9: Write architecture.md

**Files:**
- Create: `site/docs/architecture.md`

- [x] Mermaid diagram: Subscription URL → Parser → ProxyConfig → Checker → Results → Metrics/API/Dashboard
- [x] Component descriptions (one per package)
- [x] Data flow narrative
- [x] Verify: `cd site && .venv/bin/mkdocs build --strict` succeeds

### Task 10: Add Cloudflare Pages deployment config

**Files:**
- Create: `site/wrangler.toml`

- [x] Create `site/wrangler.toml` with build command and output directory per design spec
- [x] Document local development workflow in a comment or verify `.venv/bin/mkdocs serve` works
- [x] Verify: `cd site && .venv/bin/mkdocs build --strict` succeeds (final full build)

### Task 11: Verify acceptance criteria

- [x] All 7 documentation pages present and accurate to current codebase
- [x] Logo and favicon render correctly
- [x] Dark/light theme toggle works
- [x] Navigation sidebar shows all sections
- [x] Search works
- [x] Code blocks have syntax highlighting
- [x] Mermaid diagram renders in architecture page
- [x] `.venv/bin/mkdocs build --strict` passes with zero warnings
- [x] `.venv/bin/mkdocs serve` works for local preview

### Task 12: [Final] Update documentation and clean up

- [x] Update main `README.md` with link/reference to documentation site
- [x] Move this plan to `docs/plans/completed/`

## Technical Details

### MkDocs Configuration
- Theme: Material with custom color palette matching dashboard
- Features: dark/light toggle, search, code highlighting, TOC sidebar
- No navigation tabs (simple sidebar sufficient for 7 pages)

### Color Palette
- Dark primary: `#1a1a2e` bg, `#7c4dff` accent, `#00c853` success, `#ff5252` error
- Light: `#f5f5f5` bg, `#5e35b1` accent

### Deployment
- Build: `pip install -r requirements.txt && mkdocs build`
- Output: `site/site/` (MkDocs default)
- Deploy: `cd site && wrangler pages deploy site`

## Post-Completion

**Manual verification:**
- Visual review of all pages in browser (dark and light modes)
- Test on mobile viewport
- Verify Cloudflare Pages deployment works with actual account

**External system updates:**
- Configure Cloudflare Pages project (account-specific)
- Set up custom domain if desired
- Consider adding deploy script or Makefile target
