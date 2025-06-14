# ADR-0002: Pilot splitting datastore layer into separate Go packages 📦

## Status 🚦

Proposed

## Date 📅

2025-06-09

## Context 🔍

Similar to the pilot initiative to split the service layer into separate Go packages (see [ADR-0001: Pilot splitting service layer](./0001-pilot-service-layer-packages.md)), we have identified similar challenges within our Fleet datastore layer:

* 🗄️ The datastore package has grown into a monolithic structure containing all database operations across all features.
* 🔗 Tight coupling between different feature areas at the data access level makes it difficult to modify methods without risking unintended side effects.
* 🐢 Test execution times have increased significantly due to the large number of database tests running together.
* 🧩 New engineers struggle to understand which datastore methods belong to which features, as everything is mixed together in a single package.

Building on the service layer pilot (ADR-0001), we recognize that achieving true feature isolation requires addressing both the service and datastore layers. While the service layer pilot shows promising results for feature encapsulation, the shared datastore layer remains a point of coupling between features.

## Decision ✅

We have decided to **pilot splitting the Fleet datastore layer into separate Go packages**, continuing with the Android feature area that was used in the service layer pilot. This extends our move toward a **service-oriented architecture** by decoupling data access patterns alongside service logic. The Go datastore layer will be split while the MySQL schema will NOT change.

This approach allows us to:

* 🎯 Build upon the existing Android service package pilot with corresponding datastore separation.
* 📊 Measure the impact of datastore separation on test performance and development velocity.
* 🛡️ Evaluate strategies for handling cross-feature data dependencies and common datastore logic.
* 🔍 Identify patterns for database transaction management across package boundaries.

The pilot will focus on consolidating Android datastore operations into a dedicated package while maintaining compatibility with the existing shared MySQL infrastructure. The MySQL database and tables will NOT be explicitly separated by feature boundaries. This work is out of scope for this pilot.

### Additional Details

The current datastore layer faces several architectural challenges that mirror those found in the service layer:

* **Feature intermingling at the datastore level:** Database queries, models, and data access logic for all features are combined in a single datastore package, making it difficult to understand feature boundaries.
* **Testing bottlenecks:** The monolithic datastore package requires running extensive database tests even for small changes. The datastore package tests currently take the longest in CI. (Note: The service package tests technically take longer, but we split them into multiple test runs, which is a somewhat messy and inefficient pattern.)

**Pilot approach:**
The pilot will consolidate an Android-specific datastore package that:
* Encapsulates all Android-related database operations, queries, and models.
* Defines clear interfaces for data access that the Android service package can consume.
* Maintains transactional consistency when interacting with shared MySQL resources.
* Provides dedicated database testing utilities optimized for Android feature testing.

**Key considerations:**
* **Shared entities:** Strategy is required for handling entities like users, teams, and hosts that are referenced across features.
    * For hosts, the pilot will create a context-specific version of Host struct with only the fields needed for Android. The intent is to create a clear understanding of what an Android host is, have less coupling with other features, and be easier to test and maintain. This approach will be reviewed at the end of the pilot.

Directory structure:

```
server/
└── mdm/
    └── android/
        ├── service/
        └── mysql/
            ├── enterprises.go
            ├── hosts.go
            └── mysql.go
```

## Consequences 🎭

**Benefits:** ✨

* 🚀 Faster test execution by separating datastore tests into separate independent packages.
* 🎯 Clearer ownership and boundaries for feature-specific data access code.
* 📚 Better code organization, making it easier to understand and modify feature logic.
* 🧪 Improved ability to mock or stub datastore dependencies in tests.

**Drawbacks / technical debt:** ⚠️

* 🔀 Increased complexity in managing cross-feature database transactions.
* 🔄 Potential for code duplication in common database utilities across packages.
* 📊 Need for careful monitoring to ensure performance isn't compromised.
* 🥡 Potential complexity accessing shared entities (MySQL tables) referenced across features.

**Impact:** 💫

* 🏗️ Requires establishing patterns for shared database resources and transaction management.
* 📝 Need for clear documentation on package boundaries and inter-package communication.
* 🔍 May reveal hidden dependencies between features.

**Future considerations:** 🔮

* 🎯 Evaluate whether to proceed with datastore separation for other feature areas based on pilot results.
* 🏭 Consider introducing a pattern or similar abstraction to standardize data access across packages.
* 🔄 Explore options for shared database utilities package to reduce duplication.
* 📊 Develop metrics for measuring the impact on test performance and development velocity.

## Alternatives considered 🤔

**Alternative 1: Keep datastore layer monolithic**

* **Description:** Maintain the current single datastore package while only splitting the service layer.
* **Pros:** Simpler transaction management, no risk of cross-package query performance issues.
* **Cons:** Continued test performance problems, unclear feature boundaries at data level, limited benefits from service layer separation.
* **Reason not chosen:** Does not fully address the coupling issues that limit the effectiveness of service layer separation.

**Alternative 2: Multiple interfaces with single implementation**

* **Description:** Introduce interfaces in service packages but keep all implementations in the shared datastore package.
* **Pros:** Clear contracts without physical separation, easier transaction management.
* **Cons:** Doesn't address test performance issues, datastore package continues to grow.
* **Reason not chosen:** Provides only superficial separation without addressing core maintainability concerns.

**Alternative 3: Database-per-service pattern**

* **Description:** Give each service its own database, eliminating shared MySQL dependencies.
* **Pros:** Complete isolation, independent scaling.
* **Cons:** Significant infrastructure changes, complex data consistency challenges, major migration effort.
* **Reason not chosen:** Too complex and radical a change, would require extensive architectural redesign.

## References 📖

* [ADR-0001: Pilot splitting service layer into separate Go packages](./0001-pilot-service-layer-packages.md)
* [Android contributor docs](../product-groups/mdm/android-mdm.md)
