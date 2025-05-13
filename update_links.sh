#!/bin/bash

# Update links to the old directory structure
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Building-Fleet.md|docs/Contributing/getting-started/building-fleet.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Testing-and-local-development.md|docs/Contributing/getting-started/testing-and-local-development.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Run-Locally-Built-Fleetd.md|docs/Contributing/getting-started/run-locally-built-fleetd.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Migrations.md|docs/Contributing/guides/migrations.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Seeding-Data.md|docs/Contributing/guides/seeding-data.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Committing-Changes.md|docs/Contributing/guides/committing-changes.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Releasing-Fleet.md|docs/Contributing/guides/releasing-fleet.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/API-for-contributors.md|docs/Contributing/workflows/api-for-contributors.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Upgrading-go-version.md|docs/Contributing/workflows/upgrading-go-version.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Deploying-chrome-test-ext.md|docs/Contributing/workflows/deploying-chrome-test-ext.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Configuration-for-contributors.md|docs/Contributing/reference/configuration-for-contributors.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/FAQ.md|docs/Contributing/reference/faq.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/MDM.md|docs/Contributing/product-groups/mdm/mdm-overview.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/MDM-end-user-authentication.md|docs/Contributing/product-groups/mdm/mdm-end-user-authentication.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/MDM-Android.md|docs/Contributing/product-groups/mdm/android-mdm.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/MDM-Apple-account-driven-user-enrollment.md|docs/Contributing/product-groups/mdm/apple-account-driven-user-enrollment.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/MDM-Custom-SCEP-Integration.md|docs/Contributing/product-groups/mdm/custom-scep-integration.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/MDM-DigiCert-Integration.md|docs/Contributing/product-groups/mdm/digicert-integration.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/windows-mdm-glossary-and-protocol.md|docs/Contributing/product-groups/mdm/windows-mdm-glossary-and-protocol.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Understanding-host-vitals.md|docs/Contributing/product-groups/orchestration/understanding-host-vitals.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Teams.md|docs/Contributing/product-groups/orchestration/teams.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/File-carving.md|docs/Contributing/product-groups/orchestration/file-carving.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Troubleshooting-live-queries.md|docs/Contributing/guides/troubleshooting-live-queries.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Vulnerability-processing.md|docs/Contributing/guides/vulnerability-processing.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/high-level-architecture.md|docs/Contributing/architecture/high-level-architecture.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Adding-new-endpoints.md|docs/Contributing/guides/api/adding-new-endpoints.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/API-Versioning.md|docs/Contributing/reference/api-versioning.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Audit-logs.md|docs/Contributing/guides/audit-logs.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Automatically-generating-UI-component-boilerplate.md|docs/Contributing/guides/ui/generating-ui-component-boilerplate.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/design-qa-considerations.md|docs/Contributing/guides/ui/design-qa-considerations.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Enroll-hosts-with-plain-osquery.md|docs/Contributing/guides/orchestration/enroll-hosts-with-plain-osquery.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Fleet-UI-Testing.md|docs/Contributing/guides/ui/fleet-ui-testing.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/fleetctl-apply.md|docs/Contributing/guides/fleetctl-apply.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/fleetd-development-and-release-strategy.md|docs/Contributing/workflows/fleetd-development-and-release-strategy.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Infrastructure.md|docs/Contributing/architecture/infrastructure.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/MDM-custom-configuration-web-url|docs/Contributing/product-groups/mdm/custom-configuration-web-url.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Patterns-backend.md|docs/Contributing/reference/patterns-backend.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/SCIM-integration.md|docs/Contributing/guides/integration/scim-integration.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/set-up-custom-end-user-email.md|docs/Contributing/guides/mdm/set-up-custom-end-user-email.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Simulate-slow-network.md|docs/Contributing/guides/simulate-slow-network.md|g' {} \;
find /Users/luke/code/fleet -type f -name "*.md" -exec sed -i '' 's|docs/Contributing/Upcoming-activities.md|docs/Contributing/reference/upcoming-activities.md|g' {} \;

echo "Links updated successfully!"