# ADR-0004: Third-party library vendoring 📦

## Status 🚦

Proposed

## Date 📅

2025-07-30

## Context 🔍

Fleet occasionally needs to make Fleet-specific changes to external dependencies. These changes may be temporary while waiting for upstream maintainers to merge and release new versions. While we have one example following good practices (`httpsig-go`), we need to formalize this as a standard approach to ensure consistency across all vendored libraries.

Current challenges without a formal standard:

* 😕 Risk of inconsistency as new libraries are vendored
* 🔍 No guarantee future vendored libraries will include proper tracking
* 📚 Lack of documented standard for the process
* 🤷 Confusion about when to vendor vs. other approaches

The `httpsig-go` library in the `third_party` directory already follows best practices with `UPSTREAM_COMMIT` and `UPDATE_INSTRUCTIONS` files, and is slated for removal (as of 2025-07-30) once Fleet-required changes are merged upstream. This ADR formalizes these practices as the standard.

## Decision 💡

All external repositories requiring Fleet-specific changes will be vendored into the `third_party` directory following these standards:

1. **📁 Directory structure**: Each library must be copied WITHOUT the `.git` directory
2. **📄 Required files**: Each vendored library must include:
   - `UPSTREAM_COMMIT`: A file containing the exact upstream commit hash that was vendored
   - `UPDATE_INSTRUCTIONS`: A file containing detailed instructions for updating the library to the latest upstream version

3. **🔄 Update process**: The UPDATE_INSTRUCTIONS file should contain:
   - Step-by-step commands for pulling in upstream changes
   - Instructions for merging Fleet-specific changes with upstream updates
   - Commands to update the UPSTREAM_COMMIT file
   - Example commands for committing the updates

4. **📦 Migration**: Current third-party libraries copied in our repo should migrate to this format whenever they need to be updated

5. **🗑️ Removal**: When a local version is no longer needed because downstream changes have merged into upstream, the library directory should be deleted and dependencies should point to the upstream version

## Consequences 🎯

### Positive ✅
- 📏 Standardized approach for managing forked dependencies
- 📝 Clear documentation trail for tracking upstream changes
- 🚀 Easier to update vendored libraries with upstream changes
- 👩‍💻 Simplified onboarding for developers who need to update vendored dependencies
- 🎉 Clear path for eventually removing vendored libraries when changes are upstreamed

### Negative ⚠️
- 💾 Increased repository size from vendored code
- 🔧 Manual effort required to maintain UPDATE_INSTRUCTIONS
- 📅 Potential for vendored libraries to become stale if not regularly updated
- 📋 Additional process overhead when vendoring new libraries

## Alternatives considered 🤔

### Status quo: No formal standard
- **Pros**: ✅ No additional process overhead, flexibility in approach
- **Cons**: ❌ Inconsistent handling of vendored libraries, risk of missing important files like UPSTREAM_COMMIT, harder to maintain and update
- **Rejected because**: As we vendor more libraries, lack of standardization will lead to technical debt and maintenance burden

### Long-lived forks with git submodules or go.mod replace directives
- **Pros**: ✅ Standard tooling, version history preserved, easier to contribute upstream
- **Cons**: ❌ Violates Fleet's monorepo principle, requires maintaining multiple repositories, could discourage adding necessary dependencies
- **Rejected because**: As discussed in [ADR-0003](0003-fork-management.md) and [PR #31079](https://github.com/fleetdm/fleet/pull/31079), Fleet has a strong bias for keeping everything in a single repository. The cost of managing multiple repositories outweighs the benefits of this approach.

## References 📚

- 🚫 [ADR-0003: Fork management](0003-fork-management.md) (Rejected)
- 📦 Current example: `/third_party/httpsig-go/` - Already follows this standard and demonstrates the process. Scheduled for removal once upstream merges Fleet's changes.
