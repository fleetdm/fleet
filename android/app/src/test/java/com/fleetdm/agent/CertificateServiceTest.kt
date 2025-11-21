package com.fleetdm.agent

import android.app.admin.DevicePolicyManager
import android.content.Context
import android.content.Intent
import androidx.test.core.app.ApplicationProvider
import com.fleetdm.agent.scep.MockScepClient
import com.fleetdm.agent.scep.ScepClient
import com.fleetdm.agent.scep.ScepEnrollmentException
import io.mockk.*
import kotlinx.coroutines.test.runTest
import org.json.JSONObject
import org.junit.After
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.Robolectric
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import java.lang.reflect.Field

/**
 * Unit tests for CertificateService.
 *
 * Uses Robolectric for Android component testing and MockK for mocking.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [33])
class CertificateServiceTest {

    private lateinit var service: CertificateService
    private lateinit var mockScepClient: MockScepClient
    private lateinit var mockDpm: DevicePolicyManager

    @Before
    fun setup() {
        mockScepClient = MockScepClient()
        mockDpm = mockk(relaxed = true)

        // Create the service using Robolectric
        service = Robolectric.setupService(CertificateService::class.java)

        // Inject the mock SCEP client via reflection
        injectMockScepClient(service, mockScepClient)

        // Mock the system service
        every { service.getSystemService(Context.DEVICE_POLICY_SERVICE) } returns mockDpm
    }

    @After
    fun tearDown() {
        mockScepClient.reset()
        clearAllMocks()
    }

    @Test
    fun `service starts with valid CERT_DATA`() = runTest {
        val certData = createValidCertDataJson()
        val intent = Intent().apply {
            putExtra("CERT_DATA", certData.toString())
        }

        service.onStartCommand(intent, 0, 1)

        // Give coroutine time to complete
        kotlinx.coroutines.delay(100)

        // Verify SCEP client was called with correct config
        assertNotNull(mockScepClient.capturedConfig)
        assertEquals("https://scep.example.com/cgi-bin/pkiclient.exe", mockScepClient.capturedConfig?.url)
        assertEquals("secret123", mockScepClient.capturedConfig?.challenge)
        assertEquals("device-cert", mockScepClient.capturedConfig?.alias)
        assertEquals("CN=Device123,O=FleetDM", mockScepClient.capturedConfig?.subject)
    }

    @Test
    fun `service installs certificate after successful enrollment`() = runTest {
        every { mockDpm.installKeyPair(any(), any(), any(), any()) } returns true

        val certData = createValidCertDataJson()
        val intent = Intent().apply {
            putExtra("CERT_DATA", certData.toString())
        }

        service.onStartCommand(intent, 0, 1)

        // Give coroutine time to complete
        kotlinx.coroutines.delay(100)

        // Verify DPM was called to install the certificate
        verify {
            mockDpm.installKeyPair(
                null, // admin is null for delegated apps
                any(), // private key
                any(), // certificate chain
                "device-cert" // alias
            )
        }
    }

    @Test
    fun `service handles enrollment failure gracefully`() = runTest {
        mockScepClient.shouldThrowEnrollmentException = true

        val certData = createValidCertDataJson()
        val intent = Intent().apply {
            putExtra("CERT_DATA", certData.toString())
        }

        service.onStartCommand(intent, 0, 1)

        // Give coroutine time to complete
        kotlinx.coroutines.delay(100)

        // Verify DPM was NOT called since enrollment failed
        verify(exactly = 0) {
            mockDpm.installKeyPair(any(), any(), any(), any())
        }
    }

    @Test
    fun `service handles network exception gracefully`() = runTest {
        mockScepClient.shouldThrowNetworkException = true

        val certData = createValidCertDataJson()
        val intent = Intent().apply {
            putExtra("CERT_DATA", certData.toString())
        }

        service.onStartCommand(intent, 0, 1)

        // Give coroutine time to complete
        kotlinx.coroutines.delay(100)

        // Verify DPM was NOT called
        verify(exactly = 0) {
            mockDpm.installKeyPair(any(), any(), any(), any())
        }
    }

    @Test
    fun `service stops when started without CERT_DATA`() {
        val intent = Intent() // No CERT_DATA extra

        val result = service.onStartCommand(intent, 0, 1)

        // Service should stop itself
        assertEquals(android.app.Service.START_NOT_STICKY, result)
    }

    @Test
    fun `service parses custom key length and signature algorithm`() = runTest {
        val certData = JSONObject().apply {
            put("scep_url", "https://scep.example.com/cgi-bin/pkiclient.exe")
            put("challenge", "secret123")
            put("alias", "device-cert")
            put("subject", "CN=Device123,O=FleetDM")
            put("key_length", 4096)
            put("signature_algorithm", "SHA512withRSA")
        }

        val intent = Intent().apply {
            putExtra("CERT_DATA", certData.toString())
        }

        service.onStartCommand(intent, 0, 1)

        // Give coroutine time to complete
        kotlinx.coroutines.delay(100)

        // Verify config was parsed correctly
        assertEquals(4096, mockScepClient.capturedConfig?.keyLength)
        assertEquals("SHA512withRSA", mockScepClient.capturedConfig?.signatureAlgorithm)
    }

    @Test
    fun `service uses default values when optional parameters missing`() = runTest {
        val certData = JSONObject().apply {
            put("scep_url", "https://scep.example.com/cgi-bin/pkiclient.exe")
            put("challenge", "secret123")
            put("alias", "device-cert")
            put("subject", "CN=Device123,O=FleetDM")
            // key_length and signature_algorithm not provided
        }

        val intent = Intent().apply {
            putExtra("CERT_DATA", certData.toString())
        }

        service.onStartCommand(intent, 0, 1)

        // Give coroutine time to complete
        kotlinx.coroutines.delay(100)

        // Verify defaults were used
        assertEquals(2048, mockScepClient.capturedConfig?.keyLength)
        assertEquals("SHA256withRSA", mockScepClient.capturedConfig?.signatureAlgorithm)
    }

    // Helper functions

    private fun createValidCertDataJson(): JSONObject {
        return JSONObject().apply {
            put("scep_url", "https://scep.example.com/cgi-bin/pkiclient.exe")
            put("challenge", "secret123")
            put("alias", "device-cert")
            put("subject", "CN=Device123,O=FleetDM")
        }
    }

    private fun injectMockScepClient(service: CertificateService, mockClient: ScepClient) {
        try {
            val field: Field = service.javaClass.getDeclaredField("scepClient")
            field.isAccessible = true
            field.set(service, mockClient)
        } catch (e: Exception) {
            throw RuntimeException("Failed to inject mock SCEP client", e)
        }
    }
}
