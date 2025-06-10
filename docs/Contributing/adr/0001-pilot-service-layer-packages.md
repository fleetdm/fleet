# ADR-0001: Pilot splitting service layer into separate Go packages 📦

## Status 🚦

Proposed

## Date 📅

2025-06-09

## Context 🔍

Over time, our Fleet service layer has become larger and more complex, leading to several pain points:

* 🐌 Slower feature development and longer bug resolution cycles.
* 🤯 Difficulties in onboarding new engineers due to tightly coupled components within the service layer.
* 💥 Increased risk that changes in one service area inadvertently affect others.
* 🧪 Challenges in testing and maintaining a growing service package.
  * i.e., test run times, IDEs having difficulty indexing large Go packages.

The need for improved maintainability and organizational scalability is driving us to consider splitting the service layer into separate Go packages. However, concerns about risk and uncertainty have delayed broader adoption. To address these concerns, a pilot was completed for a contained portion of the Fleet service layer, allowing us to evaluate the benefits and drawbacks in a lower-risk context.

## Decision ✅

We have decided to **pilot splitting the Fleet service layer into separate Go packages**, focusing on a representative, self-contained feature area (Android). This is a move toward a **service-oriented architecture**, where the major services of Fleet are decoupled from each other as much as possible. The pilot work is in progress and will ready for review and evaluation after Android support is added for configuration profiles and software.

This approach was selected because it allows us to:

* 🛡️ Reduce risk by limiting scope to a new area within the service layer.
* 📊 Collect data on development speed, code quality, and engineering challenges.
* 🎯 Provide a real-world example for engineering leadership and the broader team.

Feedback and lessons learned will inform a potential rollout to split additional areas of the service layer into separate packages.

### Additional Details

The current Fleet server is structured using a **layered architecture** with clearly defined frontend, service, and datastore layers. While this architecture is a common and pragmatic starting point for modern software projects, over time, it has led to several challenges:

* **Feature intermingling:** Features across the application are tightly coupled and intermingled within the service and datastore layers. There is little separation by domain or feature, making the codebase harder to reason about.
* **Growing package size:** The service and datastore packages have grown significantly, becoming large monoliths in themselves. This increase in size has introduced practical bottlenecks for engineers:
  * **Slow CI and local test runs:** The large and tightly coupled nature of these packages means that even small changes can trigger long test suites, slowing down both continuous integration (CI) pipelines and local developer workflows.
  * **IDE performance:** The size and complexity of these core packages have begun to negatively impact IDE features such as code indexing, autocomplete, and navigation.
* **Risk of unintended breakage:** Because features are not well isolated, changes to one part of the service or datastore layer can inadvertently break unrelated features. This makes it difficult to ensure correctness without extensive, time-consuming manual QA of the entire application.

**Pilot approach:**
The pilot introduces an Android-specific Go package within the service layer with its own dedicated handler and service implementations. This package aims to better encapsulate feature logic, providing a clearer boundary and reducing the risk of cross-feature impact. While the new package still interacts with the shared datastore layer and MySQL database, resulting in some necessary coupling at the data layer, the service package keeps its logic as focused and decoupled as possible. The cross-cutting concerns, such as authentication/authorization, are also shared.

Directory structure:

```
server/
└── mdm/
    └── android/
        └── service/
            ├── endpoint_utils.go
            ├── handler.go
            ├── pubsub.go
            └── service.go
```

## Consequences 🎭

**Benefits:** ✨

* 📈 Enables evidence-based evaluation of splitting the service layer into separate packages.
* 📚 Provides documentation, best practices, and architectural patterns for future package separation efforts.
* 🚀 May improve onboarding, feature delivery, and code reliability in the pilot area.

**Drawbacks / technical debt:** ⚠️

* 🔀 Inconsistent architecture across the service layer as new and old patterns coexist.
* 🔧 Possible need to refactor Android service package if a different architecture is chosen later.
* ⭕️ Code that has bidirectional dependencies will need careful refactoring to avoid circular dependencies.

**Impact:** 💫

* 🌊 Minimal disruption to current workflows, as changes are isolated to the service layer of a specific feature.
* 🏗️ Sets a precedent and provides a template for future service package separation.

**Future considerations:** 🔮

* 🔬 Evaluate pilot results after initial Android features (profiles, software) are implemented.
  * Learn which functionality is difficult to separate and will require shared packages.
* 🤝 Decide whether to expand service layer package separation based on metrics and team feedback.
* 📝 Document lessons learned and adjust patterns as needed.

## Alternatives considered 🤔

**Alternative 1: Maintain status quo**

* **Description:** Continue development without splitting the service layer into separate packages.
* **Pros:** No disruption or learning curve.
* **Cons:** Existing pain points persist; future refactoring becomes more difficult as codebase grows.
* **Reason not chosen:** Does not address underlying maintainability and scalability issues.

**Alternative 2: Incremental refactoring without a formal pilot**

* **Description:** Gradually split service code into packages "as we go," without explicit pilot or metrics.
* **Pros:** Lower disruption, flexible.
* **Cons:** Difficult to measure impact, inconsistent adoption, inconsistent patterns, lack of clear outcomes.
* **Reason not chosen:** Does not provide enough evidence to inform a broader decision or demonstrate value to stakeholders.

**Alternative 3: Hexagonal architecture (ports and adapters)**

* **Description:** Structure the application around core business logic, with well-defined interfaces (ports) for communication with external systems (adapters), such as the UI, network, or database.
* **Pros:** Strong isolation of business logic, improved testability, flexibility to swap out external dependencies.
* **Cons:** Adds additional layers and abstraction, may increase development overhead for simpler features, potential learning curve.
* **Reason not chosen:** While hexagonal architecture offers valuable separation of concerns, the service-oriented architecture offers more immediate incremental benefits with less overhead and a lower barrier to adoption in the existing codebase.

*Note:* A move to a service-oriented architecture does not prevent us from also using hexagonal architecture where needed.

## References 📖

* [Android contributor docs](../product-groups/mdm/android-mdm.md)
