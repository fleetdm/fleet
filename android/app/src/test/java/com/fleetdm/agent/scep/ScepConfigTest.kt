package com.fleetdm.agent.scep

import org.junit.Assert.*
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
            subject = "CN=Device123,O=FleetDM"
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
            signatureAlgorithm = "SHA512withRSA"
        )

        assertEquals(4096, config.keyLength)
        assertEquals("SHA512withRSA", config.signatureAlgorithm)
    }

    @Test(expected = IllegalArgumentException::class)
    fun `blank url throws exception`() {
        ScepConfig(
            url = "",
            challenge = "secret",
            alias = "cert",
            subject = "CN=Test"
        )
    }

    @Test(expected = IllegalArgumentException::class)
    fun `blank challenge throws exception`() {
        ScepConfig(
            url = "https://scep.example.com",
            challenge = "",
            alias = "cert",
            subject = "CN=Test"
        )
    }

    @Test(expected = IllegalArgumentException::class)
    fun `blank alias throws exception`() {
        ScepConfig(
            url = "https://scep.example.com",
            challenge = "secret",
            alias = "",
            subject = "CN=Test"
        )
    }

    @Test(expected = IllegalArgumentException::class)
    fun `blank subject throws exception`() {
        ScepConfig(
            url = "https://scep.example.com",
            challenge = "secret",
            alias = "cert",
            subject = ""
        )
    }

    @Test(expected = IllegalArgumentException::class)
    fun `key length below 1024 throws exception`() {
        ScepConfig(
            url = "https://scep.example.com",
            challenge = "secret",
            alias = "cert",
            subject = "CN=Test",
            keyLength = 512
        )
    }

    @Test
    fun `minimum key length 1024 is accepted`() {
        val config = ScepConfig(
            url = "https://scep.example.com",
            challenge = "secret",
            alias = "cert",
            subject = "CN=Test",
            keyLength = 1024
        )

        assertEquals(1024, config.keyLength)
    }

    @Test
    fun `config equality works correctly`() {
        val config1 = ScepConfig(
            url = "https://scep.example.com",
            challenge = "secret",
            alias = "cert",
            subject = "CN=Test"
        )

        val config2 = ScepConfig(
            url = "https://scep.example.com",
            challenge = "secret",
            alias = "cert",
            subject = "CN=Test"
        )

        assertEquals(config1, config2)
        assertEquals(config1.hashCode(), config2.hashCode())
    }
}
