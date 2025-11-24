package com.fleetdm.agent.scep

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Unit tests for SCEP exception hierarchy.
 * Verifies exception types, messages, and cause handling.
 */
class ScepExceptionTest {

    @Test
    fun `ScepEnrollmentException can be created with message`() {
        val exception = ScepEnrollmentException("Enrollment failed")

        assertEquals("Enrollment failed", exception.message)
        assertNull(exception.cause)
        assertTrue(exception is ScepException)
    }

    @Test
    fun `ScepEnrollmentException can be created with cause`() {
        val rootCause = IllegalStateException("Root cause")
        val exception = ScepEnrollmentException("Enrollment failed", rootCause)

        assertEquals("Enrollment failed", exception.message)
        assertEquals(rootCause, exception.cause)
    }

    @Test
    fun `ScepNetworkException can be created with message and cause`() {
        val rootCause = java.net.UnknownHostException("Host not found")
        val exception = ScepNetworkException("Network error", rootCause)

        assertEquals("Network error", exception.message)
        assertEquals(rootCause, exception.cause)
        assertTrue(exception is ScepException)
    }

    @Test
    fun `ScepCertificateException extends ScepException`() {
        val exception = ScepCertificateException("Invalid certificate")

        assertTrue(exception is ScepException)
        assertTrue(exception is Exception)
    }

    @Test
    fun `ScepKeyGenerationException extends ScepException`() {
        val exception = ScepKeyGenerationException("Key generation failed")

        assertTrue(exception is ScepException)
        assertEquals("Key generation failed", exception.message)
    }

    @Test
    fun `ScepCsrException extends ScepException`() {
        val exception = ScepCsrException("CSR creation failed")

        assertTrue(exception is ScepException)
        assertEquals("CSR creation failed", exception.message)
    }

    @Test
    @Suppress("SwallowedException", "ThrowsCount")
    fun `all exceptions are catchable as ScepException`() {
        val exceptions = listOf<ScepException>(
            ScepEnrollmentException("enrollment"),
            ScepNetworkException("network"),
            ScepCertificateException("certificate"),
            ScepKeyGenerationException("key"),
            ScepCsrException("csr"),
        )

        exceptions.forEach { exception ->
            try {
                throw exception
            } catch (e: ScepException) {
                // Successfully caught as base type - intentionally swallowed for test
                assertNotNull(e.message)
            }
        }
    }

    @Test
    fun `exceptions maintain stack trace information`() {
        val rootCause = Exception("Root")
        val exception = ScepNetworkException("Network failed", rootCause)

        val stackTrace = exception.stackTrace
        assertNotNull(stackTrace)
        assertTrue(stackTrace.isNotEmpty())
    }

    @Test
    @Suppress("SwallowedException", "ThrowsCount")
    fun `exception types are distinguishable in catch blocks`() {
        var enrollmentCaught = false
        var networkCaught = false
        var certificateCaught = false

        try {
            throw ScepEnrollmentException("test")
        } catch (e: ScepEnrollmentException) {
            // Intentionally swallowed for test - verifying exception can be caught
            enrollmentCaught = true
        }

        try {
            throw ScepNetworkException("test")
        } catch (e: ScepNetworkException) {
            // Intentionally swallowed for test - verifying exception can be caught
            networkCaught = true
        }

        try {
            throw ScepCertificateException("test")
        } catch (e: ScepCertificateException) {
            // Intentionally swallowed for test - verifying exception can be caught
            certificateCaught = true
        }

        assertTrue(enrollmentCaught)
        assertTrue(networkCaught)
        assertTrue(certificateCaught)
    }
}
