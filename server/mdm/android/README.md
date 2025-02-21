The MDM Android package attempts to decouple Android-specific service and datastore implementations from the core Fleet server code.

Any tightly coupled code that needs both the core Fleet server and the Android-specific features must live in the main server/fleet,
server/service, and server/datastore packages. Typical example are MySQL queries. Any code that implements Android-specific functionality
should live in the server/mdm/android package. For example, the common code from server/datastore package can call the android datastore
methods as needed.

This decoupled approach attempts to achieve the following goals:
- Easier to understand and find Android-specific code.
- Easier to fix Android-specific bugs and add new features.
- Easier to maintain Android-specific feature branches.
- Faster Android-specific tests, including ability to run all tests in parallel.
