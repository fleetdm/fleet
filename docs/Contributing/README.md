# Fleet contributor documentation

Welcome to the Fleet contributor documentation! This documentation is designed to help you contribute to the Fleet project.

## Getting started

- [Getting started](getting-started/README.md) - Setup, building, and testing Fleet
- [Guides](guides/README.md) - How-to guides for common tasks
- [Workflows](workflows/README.md) - Development workflows
- [Reference](reference/README.md) - API reference, configuration, etc.
- [Architecture](architecture/README.md) - Cross-cutting architecture documentation
- [ADRs](adr/README.md) - Architectural Decision Records

## Features

Docs are organized by feature. Each feature directory holds all of that feature's contributor docs (architecture, guides, and research) with a README index. Feature areas are durable across team reorgs — directories are never named after product groups, and new feature directories are created when a feature gets its first doc.

- [MDM (cross-platform)](mdm/README.md) - Profiles, disk encryption, MDM lifecycle, migrations
- [Apple MDM](apple-mdm/README.md) - macOS, iOS, and iPadOS device management
- [Windows MDM](windows-mdm/README.md) - Windows device management and Autopilot
- [Android MDM](android-mdm/README.md) - Android device management
- [Setup experience](setup-experience/README.md) - Out-of-the-box enrollment experience
- [Authentication](authentication/README.md) - End user (IdP) authentication, certificates, device identity
- [Software](software/README.md) - Software inventory, installation, and updates
- [Vulnerability management](vulnerability-management/README.md) - Vulnerability detection and reporting
- [Automations](automations/README.md) - Policy and software automations, webhooks, calendar
- [Reports](reports/README.md) - Live and scheduled queries, query packs
- [Host vitals](host-vitals/README.md) - Host detail queries
- [Fleets and access control](fleets/README.md) - Fleets (formerly "teams") and RBAC

## Contributing

If you're new to Fleet, we recommend starting with the [Getting started](getting-started/README.md) section to set up your development environment.

Once you're set up, you can explore the [Guides](guides/README.md) section to learn how to contribute to specific areas of the project.

## Architectural Decision Records (ADRs)

We use [Architectural Decision Records](adr/README.md) to document significant architectural decisions. If you're making a significant architectural change, please create an ADR to document your decision.
