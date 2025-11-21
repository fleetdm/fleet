# Integration Tests

This document explains how integration tests are configured in the Android project.

## Overview

Integration tests (tests that require external resources like SCEP servers) are **disabled by default** in local development and only run when explicitly enabled. This prevents developers from needing to set up external infrastructure for local testing.

## Running Tests

### Local Development (Default)

```bash
cd android

# Runs all tests EXCEPT integration tests
./gradlew connectedDebugAndroidTest
```

Integration tests will be **SKIPPED** (shown in yellow in test output).

### Running Integration Tests Locally

```bash
cd android

# Enable integration tests
./gradlew connectedDebugAndroidTest -PrunIntegrationTests=true

# With custom SCEP server configuration
./gradlew connectedDebugAndroidTest \
  -PrunIntegrationTests=true \
  -Pandroid.testInstrumentationRunnerArguments.scep.url=https://your-scep-server.com/scep \
  -Pandroid.testInstrumentationRunnerArguments.scep.challenge=your-challenge
```

### CI/CD

#### GitHub Actions

```yaml
- name: Run Integration Tests
  run: |
    cd android
    ./gradlew connectedDebugAndroidTest -PrunIntegrationTests=true
```

## Writing Integration Tests

### Mark Test with @IntegrationTest

```kotlin
import com.fleetdm.agent.IntegrationTest
import com.fleetdm.agent.IntegrationTestRule
import org.junit.Rule
import org.junit.Test

class MyIntegrationTest {

    @get:Rule
    val integrationTestRule = IntegrationTestRule()

    @IntegrationTest
    @Test
    fun `test with external dependency`() {
        // This test only runs when -PrunIntegrationTests=true
        // Will be skipped during normal development
    }

    @Test
    fun `regular unit test`() {
        // This test always runs
    }
}
```

### Key Points

1. **Add the @IntegrationTest annotation** to test methods that require external resources
2. **Add the IntegrationTestRule** to your test class
3. Tests without @IntegrationTest will always run
4. Tests with @IntegrationTest only run when explicitly enabled

## How It Works

### Architecture

1. **@IntegrationTest annotation** - Marks tests that require external resources
2. **IntegrationTestRule** - JUnit rule that checks for the annotation and skips tests unless enabled
3. **Gradle configuration** - Passes the flag from project property to instrumentation runner
4. **JUnit Assume** - Gracefully skips tests with a clear message

### Configuration Flow

```
./gradlew -PrunIntegrationTests=true
         ↓
build.gradle.kts detects property
         ↓
testInstrumentationRunnerArguments["runIntegrationTests"] = "true"
         ↓
IntegrationTestRule checks instrumentation argument
         ↓
If true: runs test
If false: skips test with message
```

## Test Status Messages

### When Skipped (Default)

```
com.fleetdm.agent.scep.ScepIntegrationTest > enrollment test [SKIPPED]
```

Message: `Integration tests are disabled. Run with -PrunIntegrationTests=true to enable`

### When Running

```
com.fleetdm.agent.scep.ScepIntegrationTest > enrollment test [PASSED]
```