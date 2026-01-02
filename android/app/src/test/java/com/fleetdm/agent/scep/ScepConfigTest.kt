package com.fleetdm.agent.scep

import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test

/**
 * Unit tests for ScepConfig data model.
 * Tests validation logic and default values.
 */
class ScepConfigTest {

    @Test
    fun `valid config creates successfully`() {
        val config = ScepConfig(
            url = "https://scep.example.com/cgi-bin/pkiclient.exe",
            challenge = "secret123",
            alias = "device-cert",
            subject = "CN=Device123,O=FleetDM",
        )

        assertEquals("https://scep.example.com/cgi-bin/pkiclient.exe", config.url)
        assertEquals("secret123", config.challenge)
        assertEquals("device-cert", config.alias)
        assertEquals("CN=Device123,O=FleetDM", config.subject)
        assertEquals(2048, config.keyLength) // default
        assertEquals("SHA256withRSA", config.signatureAlgorithm) // default
    }

    @Test
    fun `config with custom key length and algorithm`() {
        val config = ScepConfig(
            url = "https://scep.example.com",
            challenge = "secret",
            alias = "cert",
            subject = "CN=Test",
            keyLength = 4096,
            signatureAlgorithm = "SHA512withRSA",
        )

        assertEquals(4096, config.keyLength)
        assertEquals("SHA512withRSA", config.signatureAlgorithm)
    }

    @Test
    fun `blank required fields throw exception`() {
        data class TestCase(
            val name: String,
            val url: String = "https://scep.example.com",
            val challenge: String = "secret",
            val alias: String = "cert",
            val subject: String = "CN=Test",
        )

        val testCases = listOf(
            TestCase(name = "blank url", url = ""),
            TestCase(name = "blank challenge", challenge = ""),
            TestCase(name = "blank alias", alias = ""),
            TestCase(name = "blank subject", subject = ""),
        )

        testCases.forEach { case ->
            assertThrows("${case.name} should throw", IllegalArgumentException::class.java) {
                ScepConfig(
                    url = case.url,
                    challenge = case.challenge,
                    alias = case.alias,
                    subject = case.subject,
                )
            }
        }
    }

    @Test
    fun `key length validation`() {
        data class TestCase(val keyLength: Int, val shouldSucceed: Boolean)

        val testCases = listOf(
            TestCase(keyLength = 1024, shouldSucceed = false),
            TestCase(keyLength = 2047, shouldSucceed = false),
            TestCase(keyLength = 2048, shouldSucceed = true),
            TestCase(keyLength = 3072, shouldSucceed = true),
            TestCase(keyLength = 4096, shouldSucceed = true),
        )

        testCases.forEach { case ->
            if (case.shouldSucceed) {
                val config = ScepConfig(
                    url = "https://scep.example.com",
                    challenge = "secret",
                    alias = "cert",
                    subject = "CN=Test",
                    keyLength = case.keyLength,
                )
                assertEquals("keyLength ${case.keyLength}", case.keyLength, config.keyLength)
            } else {
                assertThrows("keyLength ${case.keyLength} should throw", IllegalArgumentException::class.java) {
                    ScepConfig(
                        url = "https://scep.example.com",
                        challenge = "secret",
                        alias = "cert",
                        subject = "CN=Test",
                        keyLength = case.keyLength,
                    )
                }
            }
        }
    }

    @Test
    fun `url scheme validation`() {
        data class TestCase(val url: String, val shouldSucceed: Boolean)

        val testCases = listOf(
            TestCase(url = "https://scep.example.com", shouldSucceed = true),
            TestCase(url = "http://scep.example.com", shouldSucceed = true),
            TestCase(url = "scep.example.com", shouldSucceed = false),
            TestCase(url = "ftp://scep.example.com", shouldSucceed = false),
        )

        testCases.forEach { case ->
            if (case.shouldSucceed) {
                val config = ScepConfig(
                    url = case.url,
                    challenge = "secret",
                    alias = "cert",
                    subject = "CN=Test",
                )
                assertEquals("url ${case.url}", case.url, config.url)
            } else {
                assertThrows("url ${case.url} should throw", IllegalArgumentException::class.java) {
                    ScepConfig(
                        url = case.url,
                        challenge = "secret",
                        alias = "cert",
                        subject = "CN=Test",
                    )
                }
            }
        }
    }

    @Test
    fun `config equality works correctly`() {
        val config1 = ScepConfig(
            url = "https://scep.example.com",
            challenge = "secret",
            alias = "cert",
            subject = "CN=Test",
        )

        val config2 = ScepConfig(
            url = "https://scep.example.com",
            challenge = "secret",
            alias = "cert",
            subject = "CN=Test",
        )

        assertEquals(config1, config2)
        assertEquals(config1.hashCode(), config2.hashCode())
    }
}
