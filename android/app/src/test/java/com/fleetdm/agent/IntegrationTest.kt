package com.fleetdm.agent

/**
 * Marker annotation for integration tests.
 *
 * Tests annotated with @IntegrationTest will only run when:
 * - The system property "runIntegrationTests" is set to "true"
 * - Or when explicitly running with -PrunIntegrationTests flag
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
 * ./gradlew test
 * ```
 *
 * CI or explicit integration tests:
 * ```
 * ./gradlew test -PrunIntegrationTests=true
 * ```
 */
@Retention(AnnotationRetention.RUNTIME)
@Target(AnnotationTarget.FUNCTION, AnnotationTarget.CLASS)
annotation class IntegrationTest
