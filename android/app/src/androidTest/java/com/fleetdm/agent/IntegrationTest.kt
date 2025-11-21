package com.fleetdm.agent

/**
 * Marker annotation for integration tests.
 *
 * Tests annotated with @IntegrationTest will only run when:
 * - The system property "runIntegrationTests" is set to "true"
 * - Or when explicitly running the integrationTest task
 *
 * Usage:
 * ```
 * @IntegrationTest
 * @Test
 * fun `my integration test`() {
 *     // test code
 * }
 * ```
 *
 * Local development (excludes integration tests):
 * ```
 * ./gradlew connectedDebugAndroidTest
 * ```
 *
 * CI or explicit integration tests:
 * ```
 * ./gradlew connectedDebugAndroidTest -PrunIntegrationTests=true
 * ```
 */
@Retention(AnnotationRetention.RUNTIME)
@Target(AnnotationTarget.FUNCTION, AnnotationTarget.CLASS)
annotation class IntegrationTest
