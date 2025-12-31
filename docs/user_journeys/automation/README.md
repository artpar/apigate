# Screenshot & GIF Automation

This directory contains documentation for automated capture of user journey screenshots and GIFs.

---

## Quick Start

```bash
# Capture all journeys
./scripts/capture-journeys.sh all

# Capture specific journey
./scripts/capture-journeys.sh j1

# Generate GIFs only (from existing screenshots)
./scripts/capture-journeys.sh --gifs-only
```

---

## Prerequisites

1. **Node.js** - For Playwright
2. **ffmpeg** - For GIF generation (optional)
3. **APIGate running** - Server at localhost:8080

Install dependencies:
```bash
cd webui
npm install
npx playwright install chromium
```

---

## Output Structure

```
docs/
├── screenshots/
│   ├── j1-setup/
│   │   ├── 01-welcome.png
│   │   ├── 02-upstream.png
│   │   └── ...
│   ├── j2-plans/
│   ├── j3-monitor/
│   ├── j4-config/
│   ├── j5-onboarding/
│   ├── j6-api-access/
│   ├── j7-usage/
│   ├── j8-upgrade/
│   ├── j9-docs/
│   └── errors/
├── gifs/
│   ├── j1-setup-wizard.gif
│   ├── j5-signup.gif
│   └── ...
└── user_journeys/
```

---

## Naming Conventions

### Screenshots
```
{journey}-{step}-{description}.png

Examples:
j1-setup/01-welcome.png
j5-onboarding/03-filled-form.png
errors/e1-invalid-key.png
```

### GIFs
```
{journey}-{description}.gif

Examples:
j1-setup-wizard.gif
j6-create-key.gif
j8-upgrade.gif
```

---

## Capture Configuration

Each journey has capture points defined in its documentation. The Playwright test reads these and captures at each step.

Example from j1-first-time-setup.md:
```yaml
steps:
  - name: welcome
    url: /setup
    wait: networkidle

  - name: upstream-filled
    actions:
      - fill: input[name="upstream_url"]
        value: "https://api.example.com"
```

---

## GIF Parameters

| Parameter | Value | Notes |
|-----------|-------|-------|
| Frame rate | 0.5 fps | 2 seconds per frame |
| Resolution | 1280x720 | Standard viewport |
| Quality | 128 colors | Balance size/quality |
| Max duration | 15 seconds | Keep under 2MB |

---

## Maintenance

### When to Recapture

1. UI changes (layout, styling)
2. New features in journey
3. Error message changes
4. Branding updates

### Automated Triggers

Consider running capture:
- On PRs with UI changes
- Before releases
- After accessibility improvements

---

## Troubleshooting

### Server Not Running
```
Error: APIGate server is not running at localhost:8080
```
Start the server: `./apigate serve`

### Playwright Not Installed
```
Error: npx not found
```
Install Node.js and run `npm install` in webui/

### ffmpeg Not Found
```
Warning: ffmpeg not found. GIF generation will be skipped.
```
Install ffmpeg: `brew install ffmpeg` or `apt install ffmpeg`

---

## Related

- [User Journeys Overview](../README.md)
- [Capture Script](../../../scripts/capture-journeys.sh)
- [Playwright Tests](../../../webui/e2e/capture-journeys.spec.ts)
