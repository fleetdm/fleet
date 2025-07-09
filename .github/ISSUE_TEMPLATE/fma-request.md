---
name: 📦 New Fleet-maintained app
about: Request to add an app to the Fleet-maintained app catalog
title: 'New FMA: <App Name>'
labels: ':release,#g-software,fma'
assignees: 'marko-lisica'

---

### Requestor

- Application name: TODO
- Application platform: TODO (macOS/Windows)

---

### Validation

- [ ] The following outputs are generated
        - `/outputs/<app-name>/darwin.json` created
        - `/outputs/apps.json` updated
- [ ] Manifest name matches osquery `app.name` (macOS) or `programs.name` (Windows)
- [ ] Manifest version scheme matches osquery `app.short_bundle_version` (macOS) or `programs.version` (Windows) version scheme
- [ ] Manifest `unique_identifier` matches osquery `app.bundle_identifier` (macOS only)

### QA

- [ ] App adds successfully to team's library
- [ ] App installs successfully on host
- [ ] App opens successfully on host
- [ ] App uninstalls successfully on host

### Icon

- [ ] Icon added to Figma
- [ ] Icon added to Fleet
- [ ] Correct icon appears in the app catalog