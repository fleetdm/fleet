## name: New Fleet-maintained app (FMA)
about: Request a new FMA for macOS or Windows
title: "New Fleet-maintained app: [APP NAME]"
labels: fma, :help-solutions-consulting
assignees: ''

## Description

### App details

- **Application name:** [e.g., Mozilla Firefox]
- **Application platform:** [macOS/Windows]

---

## Validation checklist

- [ ] `/outputs/<app-name>/<platform>.json` created
- [ ] `/outputs/apps.json` updated
- [ ] Manifest name matches osquery `app.name` (macOS) or `programs.name` (Windows)
- [ ] Manifest version scheme matches osquery `app.short_bundle_version` (macOS) or `programs.version` (Windows)
- [ ] Manifest `unique_identifier` matches osquery `app.bundle_identifier` (macOS only)

---

## QA checklist

- [ ] App adds successfully to team's library
- [ ] App installs successfully on host
- [ ] App opens successfully on host
- [ ] App uninstalls successfully on host

---

## Icon

- [ ] Icon added to Fleet
- [ ] Correct icon appears in the app catalog

---

## Additional Notes

[Add any additional context, dependencies, or special instructions here.]
