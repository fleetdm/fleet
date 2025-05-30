# ADR-0001: Pilot modularization of Fleet codebase

## Status

Proposed

## Date

2025-05-30

## Context

Over time, our Fleet codebase has become larger and more complex, leading to several pain points:

* Slower feature development and longer bug resolution cycles.
* Difficulties in onboarding new engineers due to tightly coupled components.
* Increased risk that changes in one area inadvertently affect others.
* Challenges in testing and maintaining a growing codebase.
  * i.e., test run times, IDEs having difficulty indexing large Go packages.

The need for improved maintainability and organizational scalability is driving us to consider modularization. However, concerns about risk and uncertainty have delayed broader adoption. To address these concerns, a pilot modularization was completed for a contained portion of the Fleet codebase, allowing us to evaluate the benefits and drawbacks in a lower-risk context.

## Decision

We have decided to **pilot modularization on the Fleet codebase**, focusing on a representative, self-contained feature area (Android). The pilot work is in progress and will ready for review and evaluation after Android support is added for configuration profiles and software.

This approach was selected because it allows us to:

* Reduce risk by limiting scope to a new area.
* Collect data on development speed, code quality, and engineering challenges.
* Provide a real-world example for engineering leadership and the broader team.

Feedback and lessons learned will inform a potential rollout to additional areas of the codebase.

### Additional Details

The current Fleet server is structured using a **layered architecture** with clearly defined frontend, service, and datastore layers. While this architecture is a common and pragmatic starting point for modern software projects, over time, it has led to several challenges:

* **Feature intermingling:** Features across the application are tightly coupled and intermingled within the service and datastore layers. There is little separation by domain or feature, making the codebase harder to reason about.
* **Growing package size:** The service and datastore packages have grown significantly, becoming large monoliths in themselves. This increase in size has introduced practical bottlenecks for engineers:
  * **Slow CI and local test runs:** The large and tightly coupled nature of these packages means that even small changes can trigger long test suites, slowing down both continuous integration (CI) pipelines and local developer workflows.
  * **IDE performance:** The size and complexity of these core packages have begun to negatively impact IDE features such as code indexing, autocomplete, and navigation.
* **Risk of unintended breakage:** Because features are not well isolated, changes to one part of the service or datastore layer can inadvertently break unrelated features. This makes it difficult to ensure correctness without extensive, time-consuming manual QA of the entire application.

**Pilot approach:**
The pilot introduces an Android-specific module with its own dedicated service and datastore packages. This module aims to better encapsulate feature logic and data access, providing a clearer boundary and reducing the risk of cross-feature impact. While the new module still relies on a shared MySQL database, resulting in some necessary coupling at the data layer, the internal architecture of the module keeps service and datastore logic as decoupled as possible.

**Measurement and Continuous Improvement:**
To accurately assess the impact of this and future architectural change, we should establish and monitor key engineering metrics, specifically:

* [Time to fix](https://github.com/fleetdm/fleet/issues/29140): How long it takes to resolve defects.

## Consequences

**Benefits:**

* Enables evidence-based evaluation of modularization’s impact.
* Provides documentation, best practices, and architectural patterns for future modularization efforts.
* May improve onboarding, feature delivery, and code reliability in the pilot area.

**Drawbacks / technical debt:**

* Inconsistent architecture across the codebase as new and old patterns coexist.
* Possible need to refactor Android module if a different architecture is chosen later.

**Impact:**

* Minimal disruption to current workflows, as changes are isolated.
* Sets a precedent and provides a template for future modularization.

**Future considerations:**

* Evaluate pilot results after initial Android features (profiles, software) are implemented.
* Decide whether to expand modularization based on metrics and team feedback.
* Document lessons learned and adjust patterns as needed.

## Alternatives considered

**Alternative 1: Maintain status quo**

* **Description:** Continue development without pursuing modularization.
* **Pros:** No disruption or learning curve.
* **Cons:** Existing pain points persist; future refactoring becomes more difficult as codebase grows.
* **Reason not chosen:** Does not address underlying maintainability and scalability issues.

**Alternative 2: Incremental refactoring without a formal pilot**

* **Description:** Gradually modularize code "as we go," without explicit pilot or metrics.
* **Pros:** Lower disruption, flexible.
* **Cons:** Difficult to measure impact, inconsistent adoption, inconsistent patterns, lack of clear outcomes.
* **Reason not chosen:** Does not provide enough evidence to inform a broader decision or demonstrate value to stakeholders.

**Alternative 3: Service-based (microservice) architecture**

* **Description:** Break down the application into independently deployable services, each handling a specific feature or domain, communicating over defined APIs (or via events).
* **Pros:** Strong boundaries, clear separation of concerns, independent deployability, and scalability.
* **Cons:** Significant infrastructure overhead, more difficult to deploy on-prem, much higher complexity, and potentially overengineered for our use case.
* **Reason not chosen:** The overhead and complexity of a service-based approach are not justified for our application, where process and deployment boundaries are currently unnecessary.

*Note:* A modular monolith architecture is easier to transition to a service-based architecture in the future. For example, if we run into scalability issues with our future SaaS offering.

**Alternative 4: Hexagonal architecture (ports and adapters)**

* **Description:** Structure the application around core business logic, with well-defined interfaces (ports) for communication with external systems (adapters), such as the UI, network, or database.
* **Pros:** Strong isolation of business logic, improved testability, flexibility to swap out external dependencies.
* **Cons:** Adds additional layers and abstraction, may increase development overhead for simpler features, potential learning curve.
* **Reason not chosen:** While hexagonal architecture offers valuable separation of concerns, a modular approach provides more immediate, incremental benefits with less overhead and a lower barrier to adoption in the existing codebase.

*Note:* A module in Fleet's codebase could be refactored to use hexagonal architecture by moving the core business logic from service/datastore layers into its own package. A move to a modular monolith architecture does not prevent us from using hexagonal architecture in the future.

## References

* [Android contributor docs](../product-groups/mdm/android-mdm.md)
