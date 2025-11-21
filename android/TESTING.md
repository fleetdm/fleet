# Testing Guide for Android SCEP Implementation

This guide covers testing strategies for the SCEP (Simple Certificate Enrollment Protocol) implementation in the Fleet Android agent.

## Table of Contents

1. [Test Structure](#test-structure)
2. [Running Tests](#running-tests)
3. [Test Categories](#test-categories)
4. [Writing New Tests](#writing-new-tests)
5. [Integration Testing](#integration-testing)
6. [Troubleshooting](#troubleshooting)

---

## Test Structure

```
./android/app/src/test/java/com/fleetdm/agent/
├── CertificateServiceTest.kt           # Tests for the service
└── scep/
    ├── ScepConfigTest.kt                # Data model validation tests
    ├── ScepClientImplTest.kt            # SCEP client implementation tests
    └── MockScepClient.kt                # Mock implementation for testing
```

---

## Running Tests

> **Integration Tests:** By default, integration tests are SKIPPED during local development. See [INTEGRATION_TESTS.md](./INTEGRATION_TESTS.md) for details on running them in CI or locally.

### Run All Unit Tests

```bash
cd android
./gradlew test
```

### Run All Android Tests (Excludes Integration Tests)

```bash
cd android
./gradlew connectedDebugAndroidTest
```

### Run All Tests Including Integration Tests

```bash
cd android
./gradlew test connectedDebugAndroidTest -PrunIntegrationTests=true
```

### Run Specific Test Class

```bash
./gradlew test --tests "com.fleetdm.agent.scep.ScepConfigTest"
```

### Run Single Test Method

```bash
./gradlew test --tests "com.fleetdm.agent.scep.ScepConfigTest.valid config creates successfully"
```

### Run Tests with Coverage

```bash
./gradlew testDebugUnitTest jacocoTestReport
```

Coverage reports will be generated in:
`./android/app/build/reports/jacoco/testDebugUnitTest/html/index.html`

### Run Tests in Android Studio

1. Right-click on test file or test method
2. Select "Run 'TestName'"
3. View results in the Run tool window

---

## Test Categories

### 1. Unit Tests (Fast, No Android Dependencies)

#### ScepConfigTest
Tests data model validation and default values.

**What it tests:**
- Valid configuration creation
- Default values (keyLength, signatureAlgorithm)
- Validation rules (blank fields, minimum key length)
- Equality and hashCode

**Example:**
```kotlin
@Test
fun `valid config creates successfully`() {
    val config = ScepConfig(
        url = "https://scep.example.com",
        challenge = "secret",
        alias = "cert",
        subject = "CN=Device"
    )
    assertEquals(2048, config.keyLength) // default
}
```

### 2. Component Tests (With Android Components)

#### CertificateServiceTest
Tests the Android Service using Robolectric.

**What it tests:**
- Service lifecycle
- Intent handling
- SCEP enrollment flow
- Certificate installation via DevicePolicyManager
- Error handling
- JSON parsing

**Example:**
```kotlin
@Test
fun `service installs certificate after successful enrollment`() = runTest {
    val certData = JSONObject().apply {
        put("scep_url", "https://scep.example.com")
        put("challenge", "secret")
        put("alias", "cert")
        put("subject", "CN=Device")
    }

    val intent = Intent().apply {
        putExtra("CERT_DATA", certData.toString())
    }

    service.onStartCommand(intent, 0, 1)

    verify {
        mockDpm.installKeyPair(null, any(), any(), "cert")
    }
}
```

### 3. Integration Tests (With Real Dependencies)

#### ScepClientImplTest
Tests error handling in ScepClientImpl.

**What it tests:**
- Invalid URL handling
- Invalid subject name handling
- Network error handling
- Initialization

**Note:** Full integration tests with a real SCEP server are recommended for CI/CD pipelines.

---

## Writing New Tests

### Testing with MockScepClient

The `MockScepClient` provides configurable behavior for testing:

```kotlin
@Test
fun `test enrollment failure handling`() = runTest {
    val mockClient = MockScepClient()
    mockClient.shouldThrowEnrollmentException = true

    try {
        mockClient.enroll(config)
        fail("Expected exception")
    } catch (e: ScepEnrollmentException) {
        // Expected
    }
}
```

### Available Mock Configurations

```kotlin
mockScepClient.shouldSucceed = false
mockScepClient.shouldThrowEnrollmentException = true
mockScepClient.shouldThrowNetworkException = true
mockScepClient.shouldThrowCertificateException = true
mockScepClient.enrollmentDelay = 1000L // milliseconds
```

### Testing Coroutines

Use `kotlinx-coroutines-test` for testing suspend functions:

```kotlin
import kotlinx.coroutines.test.runTest

@Test
fun `test async operation`() = runTest {
    val result = scepClient.enroll(config)
    assertNotNull(result)
}
```

### Mocking Android Components

Use MockK for mocking Android system services:

```kotlin
import io.mockk.*

@Test
fun `test DevicePolicyManager interaction`() {
    val mockDpm = mockk<DevicePolicyManager>()
    every { mockDpm.installKeyPair(any(), any(), any(), any()) } returns true

    // Test code here

    verify {
        mockDpm.installKeyPair(null, any(), any(), "cert-alias")
    }
}
```

---

## Integration Testing

### Testing with a Real SCEP Server

For full integration testing, you'll need:

1. **Test SCEP Server**
   - Microsoft NDES (Network Device Enrollment Service)
   - OpenXPKI
   - Ejbca
   - Or a mock SCEP server

2. **Test Configuration**

Create a test configuration in your test resources:

```kotlin
// src/androidTest/java/com/fleetdm/agent/scep/ScepIntegrationTest.kt

@RunWith(AndroidJUnit4::class)
class ScepIntegrationTest {

    @Test
    fun `enroll with real SCEP server`() = runTest {
        val config = ScepConfig(
            url = "https://test-scep-server.example.com/scep",
            challenge = getTestChallenge(),
            alias = "test-cert-${System.currentTimeMillis()}",
            subject = "CN=TestDevice,O=FleetDM"
        )

        val client = ScepClientImpl()
        val result = client.enroll(config)

        assertNotNull(result.privateKey)
        assertTrue(result.certificateChain.isNotEmpty())
    }

    private fun getTestChallenge(): String {
        // Load from test configuration
        return InstrumentationRegistry.getArguments()
            .getString("scep.test.challenge") ?: "default-test-challenge"
    }
}
```

3. **Running Integration Tests**

```bash
./gradlew connectedAndroidTest \
  -Pandroid.testInstrumentationRunnerArguments.scep.test.challenge=your-challenge
```

### Mock SCEP Server Setup

For CI/CD, consider using a containerized mock SCEP server:

```yaml
# docker-compose.test.yml
version: '3'
services:
  scep-server:
    image: micromdm/scep:latest
    ports:
      - "8080:8080"
    environment:
      - SCEP_CHALLENGE=test-challenge-123
```

---

## Test Dependencies

Current test dependencies (already added to `build.gradle.kts`):

```kotlin
// Unit Testing
testImplementation("junit:junit:4.13.2")
testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:1.9.0")

// Mocking
testImplementation("io.mockk:mockk:1.13.13")
testImplementation("io.mockk:mockk-android:1.13.13")

// Android Testing
testImplementation("org.robolectric:robolectric:4.13")
testImplementation("androidx.test:core:1.6.1")
testImplementation("androidx.test.ext:junit:1.2.1")

// Instrumentation Testing
androidTestImplementation("androidx.test.ext:junit:1.2.1")
androidTestImplementation("androidx.test.espresso:espresso-core:3.6.1")
androidTestImplementation("io.mockk:mockk-android:1.13.13")
```

---

## Troubleshooting

### Common Issues

#### 1. BouncyCastle Provider Conflicts

**Problem:** `java.security.NoSuchAlgorithmException` or provider conflicts

**Solution:** Ensure BouncyCastle is properly initialized in tests:

```kotlin
@Before
fun setup() {
    if (Security.getProvider(BouncyCastleProvider.PROVIDER_NAME) == null) {
        Security.addProvider(BouncyCastleProvider())
    }
}
```

#### 2. Robolectric Shadow Errors

**Problem:** Android components not working in tests

**Solution:** Ensure `@RunWith(RobolectricTestRunner::class)` and SDK config:

```kotlin
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [33])
class MyTest { ... }
```

#### 3. Coroutine Tests Hanging

**Problem:** Tests using coroutines never complete

**Solution:** Use `runTest` from `kotlinx-coroutines-test`:

```kotlin
import kotlinx.coroutines.test.runTest

@Test
fun myTest() = runTest {
    // Your test code
}
```

#### 4. Reflection Failures in Tests

**Problem:** Cannot inject mocks via reflection

**Solution:** Make the field accessible and handle errors:

```kotlin
private fun injectMock(target: Any, fieldName: String, mock: Any) {
    try {
        val field = target.javaClass.getDeclaredField(fieldName)
        field.isAccessible = true
        field.set(target, mock)
    } catch (e: Exception) {
        throw RuntimeException("Failed to inject mock for $fieldName", e)
    }
}
```

### Test Debugging

Enable verbose logging in tests:

```kotlin
@Before
fun setupLogging() {
    System.setProperty("robolectric.logging", "stdout")
    System.setProperty("robolectric.logging.enabled", "true")
}
```

---

## Best Practices

1. **Test Naming:** Use descriptive names with backticks for readability
   ```kotlin
   @Test
   fun `service handles enrollment failure gracefully`() { ... }
   ```

2. **Arrange-Act-Assert:** Structure tests clearly
   ```kotlin
   // Arrange
   val config = ScepConfig(...)

   // Act
   val result = client.enroll(config)

   // Assert
   assertNotNull(result)
   ```

3. **Mock Minimally:** Only mock external dependencies, test real logic

4. **Fast Tests:** Keep unit tests under 100ms, use integration tests for slow operations

5. **Cleanup:** Reset mocks and clear state in `@After` methods
   ```kotlin
   @After
   fun tearDown() {
       mockScepClient.reset()
       clearAllMocks()
   }
   ```

6. **Test Coverage:** Aim for 80%+ coverage on business logic

---

## Continuous Integration

### GitHub Actions Example

```yaml
name: Android Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up JDK 11
        uses: actions/setup-java@v3
        with:
          java-version: '11'

      - name: Run Unit Tests
        run: |
          cd android
          ./gradlew test

      - name: Generate Test Report
        run: |
          cd android
          ./gradlew testDebugUnitTest jacocoTestReport

      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./android/app/build/reports/jacoco/testDebugUnitTest/jacocoTestReport.xml
```

---

## Additional Resources

- [Kotlin Coroutines Testing Guide](https://kotlinlang.org/api/kotlinx.coroutines/kotlinx-coroutines-test/)
- [MockK Documentation](https://mockk.io/)
- [Robolectric Documentation](http://robolectric.org/)
- [Android Testing Guide](https://developer.android.com/training/testing)
