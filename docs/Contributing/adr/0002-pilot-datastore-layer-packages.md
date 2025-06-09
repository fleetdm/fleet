# ADR-0002: Pilot splitting datastore layer into separate Go packages 📦

## Status 🚦

Proposed

## Date 📅

2025-06-09

## Context 🔍

Following the pilot initiative to split the service layer into separate Go packages (see [ADR-0001](./0001-pilot-service-layer-packages.md)), we have identified similar challenges within our Fleet datastore layer:

* 🗄️ The datastore package has grown into a monolithic structure containing all database operations across all features.
* 🔗 Tight coupling between different feature areas at the data access level makes it difficult to modify database schemas or queries without risking unintended side effects.
* 🐢 Test execution times have increased significantly due to the large number of database tests running together.
* 🧩 New engineers struggle to understand which datastore methods belong to which features, as everything is mixed together in a single package.
* 🔄 Database migrations and schema changes require careful coordination across the entire datastore package.

Building on the lessons learned from the service layer pilot (ADR-0001), we recognize that achieving true feature isolation requires addressing both the service and datastore layers. While the service layer pilot showed promising results for feature encapsulation, the shared datastore layer remains a point of coupling between features.

## Decision ✅

We have decided to **pilot splitting the Fleet datastore layer into separate Go packages**, continuing with the Android feature area that was used in the service layer pilot. This extends our move toward a **service-oriented architecture** by decoupling data access patterns alongside service logic.

This approach allows us to:

* 🎯 Build upon the existing Android service package pilot with corresponding datastore separation.
* 📊 Measure the impact of datastore separation on test performance and development velocity.
* 🛡️ Evaluate strategies for handling cross-feature data dependencies and shared database resources.
* 🔍 Identify patterns for database transaction management across package boundaries.

The pilot will focus on extracting Android-specific datastore operations into a dedicated package while maintaining compatibility with the existing shared datastore infrastructure.

### Additional Details

The current datastore layer faces several architectural challenges that mirror those found in the service layer:

* **Feature intermingling at the data level:** Database queries, models, and data access logic for all features are combined in a single datastore package, making it difficult to understand feature boundaries.
* **Shared database schema coupling:** While features may have distinct service logic, they often share database tables or have foreign key relationships, creating implicit dependencies.
* **Testing bottlenecks:** The monolithic datastore package requires running extensive database tests even for small changes, significantly impacting developer productivity.
* **Migration complexity:** Database migrations must consider the entire application's data model, making it challenging to evolve individual features independently.

**Pilot approach:**
The pilot will create an Android-specific datastore package that:
* Encapsulates all Android-related database operations, queries, and models.
* Defines clear interfaces for data access that the Android service package can consume.
* Handles Android-specific database migrations while coordinating with the central migration system.
* Maintains transactional consistency when interacting with shared database resources.
* Provides dedicated database testing utilities optimized for Android feature testing.

**Key considerations:**
* **Transaction boundaries:** Careful design needed for operations that span multiple datastore packages.
* **Shared entities:** Strategy required for handling entities like users, teams, and hosts that are referenced across features.
* **Migration coordination:** Process for ensuring database migrations across packages are applied in the correct order.
* **Performance optimization:** Approach for maintaining query performance when joins cross package boundaries.

## Consequences 🎭

**Benefits:** ✨

* 🚀 Faster test execution by running only relevant datastore tests for feature changes.
* 🎯 Clearer ownership and boundaries for feature-specific data access code.
* 🔧 Easier database schema evolution for individual features without impacting others.
* 📚 Better code organization making it easier to understand and modify feature data models.
* 🧪 Improved ability to mock or stub datastore dependencies in tests.

**Drawbacks / technical debt:** ⚠️

* 🔀 Increased complexity in managing cross-feature database transactions.
* 🔄 Potential for code duplication in common database utilities across packages.
* 📊 Need for careful performance monitoring to ensure query optimization isn't compromised.
* 🗂️ Additional overhead in coordinating database migrations across packages.
* ⚡ Risk of N+1 query problems when traversing relationships across package boundaries.

**Impact:** 💫

* 🏗️ Requires establishing patterns for shared database resources and transaction management.
* 📝 Need for clear documentation on package boundaries and inter-package communication.
* 🔍 May reveal hidden dependencies between features at the data level.

**Future considerations:** 🔮

* 🎯 Evaluate whether to proceed with datastore separation for other feature areas based on pilot results.
* 🏭 Consider introducing a repository pattern or similar abstraction to standardize data access across packages.
* 🔄 Explore options for shared database utilities package to reduce duplication.
* 📊 Develop metrics for measuring the impact on test performance and development velocity.
* 🗄️ Investigate potential for feature-specific database schemas or even separate databases in the future.

## Alternatives considered 🤔

**Alternative 1: Keep datastore layer monolithic**

* **Description:** Maintain the current single datastore package while only splitting the service layer.
* **Pros:** Simpler transaction management, no risk of cross-package query performance issues.
* **Cons:** Continued test performance problems, unclear feature boundaries at data level, limited benefits from service layer separation.
* **Reason not chosen:** Does not fully address the coupling issues that limit the effectiveness of service layer separation.

**Alternative 2: Repository pattern with single implementation**

* **Description:** Introduce repository interfaces in service packages but keep all implementations in the shared datastore package.
* **Pros:** Clear contracts without physical separation, easier transaction management.
* **Cons:** Doesn't address test performance issues, datastore package continues to grow.
* **Reason not chosen:** Provides only superficial separation without addressing core maintainability concerns.

**Alternative 3: Database-per-service pattern**

* **Description:** Give each service its own database, eliminating shared database dependencies.
* **Pros:** Complete isolation, independent scaling, true microservice architecture.
* **Cons:** Significant infrastructure changes, complex data consistency challenges, major migration effort.
* **Reason not chosen:** Too radical a change for a pilot, would require extensive architectural redesign.

**Alternative 4: CQRS (Command Query Responsibility Segregation)**

* **Description:** Separate read and write models, potentially with different datastores optimized for each.
* **Pros:** Optimized for different access patterns, better scalability, clear separation of concerns.
* **Cons:** Increased complexity, eventual consistency challenges, significant learning curve.
* **Reason not chosen:** Adds unnecessary complexity for the current pilot goals, though could be considered in the future.

## References 📖

* [ADR-0001: Pilot splitting service layer into separate Go packages](./0001-pilot-service-layer-packages.md)
* [Android contributor docs](../product-groups/mdm/android-mdm.md)
* [Fleet datastore architecture documentation](../architecture/datastore.md) *(if exists)*