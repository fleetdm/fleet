package com.fleetdm.agent

import org.junit.Assume
import org.junit.rules.TestRule
import org.junit.runner.Description
import org.junit.runners.model.Statement

/**
 * JUnit rule that skips integration tests unless explicitly enabled.
 *
 * Integration tests are only run when:
 * - The system property "runIntegrationTests" is set to "true"
 *
 * Usage in test class:
 * ```
 * @get:Rule
 * val integrationTestRule = IntegrationTestRule()
 *
 * @IntegrationTest
 * @Test
 * fun `my integration test`() {
 *     // test code
 * }
 * ```
 */
class IntegrationTestRule : TestRule {

    override fun apply(base: Statement, description: Description): Statement = object : Statement() {
        override fun evaluate() {
            val hasIntegrationAnnotation = description.getAnnotation(IntegrationTest::class.java) != null ||
                description.testClass?.getAnnotation(IntegrationTest::class.java) != null

            if (hasIntegrationAnnotation) {
                val runIntegrationTests = System.getProperty("runIntegrationTests", "false").toBoolean()

                Assume.assumeTrue(
                    "Integration tests are disabled. Run with -PrunIntegrationTests=true to enable",
                    runIntegrationTests,
                )
            }

            base.evaluate()
        }
    }
}
