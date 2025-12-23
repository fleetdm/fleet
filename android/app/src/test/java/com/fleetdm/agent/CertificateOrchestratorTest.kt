package com.fleetdm.agent

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import com.fleetdm.agent.scep.MockScepClient
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Ignore
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config
import java.security.PrivateKey
import java.security.cert.Certificate
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json

/**
 * Unit tests for CertificateOrchestrator with DataStore-based certificate tracking.
 *
 * Tests:
 * - DataStore certificate tracking (JSON storage)
 * - Mutex protection for concurrent operations
 * - Optimized API call avoidance
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [33]) // Target SDK 33 for testing
class CertificateOrchestratorTest {

    private lateinit var context: Context
    private lateinit var mockScepClient: MockScepClient
    private lateinit var mockInstaller: MockCertificateInstaller

    private val json = Json {
        ignoreUnknownKeys = true
        encodeDefaults = true
    }

    @Before
    fun setup() = runTest {
        context = RuntimeEnvironment.getApplication()
        mockScepClient = MockScepClient()
        mockInstaller = MockCertificateInstaller()

        // Clear DataStore before each test
        clearDataStore()
    }

    @After
    fun tearDown() = runTest {
        clearDataStore()
        mockScepClient.reset()
        mockInstaller.reset()
    }

    // ========== Helper Functions ==========

    private suspend fun clearDataStore() {
        context.prefDataStore.edit { preferences ->
            preferences.clear()
        }
    }

    private suspend fun getStoredCertificates(): CertificateStateMap {
        val prefs = context.prefDataStore.data.first()
        val jsonString = prefs[stringPreferencesKey("installed_certificates")] ?: return emptyMap()
        return json.decodeFromString<CertificateStateMap>(jsonString)
    }

    private suspend fun storeTestCertificateInDataStore(
        certificateId: Int,
        alias: String,
        status: CertificateStatus = CertificateStatus.INSTALLED,
        retries: Int = 0,
    ) {
        context.prefDataStore.edit { preferences ->
            val existing = preferences[stringPreferencesKey("installed_certificates")]?.let {
                json.decodeFromString<CertificateStateMap>(it)
            } ?: emptyMap()

            val certInfo = CertificateState(alias, status, retries)
            val updated = existing.toMutableMap().apply {
                put(certificateId, certInfo)
            }

            val jsonString = json.encodeToString(updated)
            preferences[stringPreferencesKey("installed_certificates")] = jsonString
        }
    }

    // ========== Mock Certificate Installer ==========

    class MockCertificateInstaller : CertificateEnrollmentHandler.CertificateInstaller {
        var shouldSucceed = true
        var wasInstallCalled = false
        var capturedAlias: String? = null
        val installedCertificates = mutableSetOf<String>()

        override fun installCertificate(alias: String, privateKey: PrivateKey, certificateChain: Array<Certificate>): Boolean {
            wasInstallCalled = true
            capturedAlias = alias
            if (shouldSucceed) {
                installedCertificates.add(alias)
            }
            return shouldSucceed
        }

        fun hasKeyPair(alias: String): Boolean = installedCertificates.contains(alias)

        fun reset() {
            shouldSucceed = true
            wasInstallCalled = false
            capturedAlias = null
            installedCertificates.clear()
        }
    }

    // ========== Test Category 1: DataStore Certificate Tracking ==========

    @Test
    fun `storeCertificateInstallation stores certificate in DataStore`() = runTest {
        // Act
        CertificateOrchestrator.markCertificateInstalled(context, 123, "test-cert-1")

        // Assert
        val stored = getStoredCertificates()
        assertEquals(1, stored.size)
        assertEquals("test-cert-1", stored[123]?.alias)
    }

    @Test
    fun `storeCertificateInstallation handles multiple certificates`() = runTest {
        // Act
        CertificateOrchestrator.markCertificateInstalled(context, 123, "cert-1")
        CertificateOrchestrator.markCertificateInstalled(context, 456, "cert-2")
        CertificateOrchestrator.markCertificateInstalled(context, 789, "cert-3")

        // Assert
        val stored = getStoredCertificates()
        assertEquals(3, stored.size)
        assertEquals("cert-1", stored[123]?.alias)
        assertEquals("cert-2", stored[456]?.alias)
        assertEquals("cert-3", stored[789]?.alias)
    }

    @Test
    fun `storeCertificateInstallation updates existing certificate`() = runTest {
        // Arrange
        CertificateOrchestrator.markCertificateInstalled(context, 123, "old-alias")

        // Act - Update the same certificate ID
        CertificateOrchestrator.markCertificateInstalled(context, 123, "new-alias")

        // Assert
        val stored = getStoredCertificates()
        assertEquals(1, stored.size) // Should not duplicate
        assertEquals("new-alias", stored[123]?.alias) // Should be updated
    }

    @Test
    fun `getCertificateAlias returns null for non-existent certificate`() = runTest {
        // Act
        val alias = CertificateOrchestrator.getCertificateAlias(context, 999)

        // Assert
        assertNull(alias)
    }

    @Test
    fun `getCertificateAlias retrieves stored certificate`() = runTest {
        // Arrange
        CertificateOrchestrator.markCertificateInstalled(context, 456, "my-cert")

        // Act
        val alias = CertificateOrchestrator.getCertificateAlias(context, 456)

        // Assert
        assertEquals("my-cert", alias)
    }

    @Test
    fun `getInstalledCertificates returns empty map when DataStore is empty`() = runTest {
        // Act
        val certificates = CertificateOrchestrator.getCertificateStates(context)

        // Assert
        assertTrue(certificates.isEmpty())
    }

    @Test
    fun `getInstalledCertificates recovers from malformed JSON`() = runTest {
        // Arrange: Manually corrupt DataStore with invalid JSON
        context.prefDataStore.edit { preferences ->
            preferences[stringPreferencesKey("installed_certificates")] = "{ invalid json }"
        }

        // Act: Should not throw, returns empty map
        val certificates = CertificateOrchestrator.getCertificateStates(context)

        // Assert
        assertTrue(certificates.isEmpty())

        // Verify we can still store new certificates after recovery
        CertificateOrchestrator.markCertificateInstalled(context, 111, "recovered-cert")
        val stored = getStoredCertificates()
        assertEquals(1, stored.size)
        assertEquals("recovered-cert", stored[111]?.alias)
    }

    // ========== Test Category 2: Optimized API Call Avoidance ==========

    @Ignore("Requires DevicePolicyManager mocking - TODO: redesign test or add DI")
    @Test
    fun `isCertificateIdInstalled returns true when certificate tracked and in keystore`() = runTest {
        // Arrange
        val certificateId = 123
        val alias = "device-cert"

        storeTestCertificateInDataStore(certificateId, alias)
        mockInstaller.installedCertificates.add(alias)

        // Act
        val result = CertificateOrchestrator.isCertificateIdInstalled(context, certificateId)

        // Assert
        assertTrue(result)
    }

    @Test
    fun `isCertificateIdInstalled returns false when certificate not in DataStore`() = runTest {
        // Act
        val result = CertificateOrchestrator.isCertificateIdInstalled(context, 999)

        // Assert
        assertFalse(result)
    }

    @Test
    fun `isCertificateIdInstalled returns false when certificate tracked but missing from keystore`() = runTest {
        // Arrange: Store in DataStore but not in keystore
        val certificateId = 456
        val alias = "missing-cert"

        storeTestCertificateInDataStore(certificateId, alias)
        // Don't add to mockInstaller.installedCertificates

        // Note: isCertificateInstalled() uses real DevicePolicyManager, not mockInstaller
        // So this test verifies DataStore logic only. The keystore check will return false
        // because the certificate doesn't actually exist in Robolectric's shadow DPM.

        // Act
        val result = CertificateOrchestrator.isCertificateIdInstalled(context, certificateId)

        // Assert
        assertFalse(result)
    }

    // ========== Test Category 3: Mutex Protection (Concurrency) ==========

    @Test
    fun `concurrent certificate storage does not lose data`() = runTest {
        // Arrange: 10 different certificate IDs
        val certificateIds = (1..10).toList()

        // Act: Store all in parallel
        val jobs = certificateIds.map { certId ->
            launch {
                CertificateOrchestrator.markCertificateInstalled(
                    context,
                    certId,
                    "cert-$certId",
                )
            }
        }
        jobs.forEach { it.join() }

        // Assert: All 10 certificates should be stored
        val stored = getStoredCertificates()
        assertEquals("All 10 certificates should be stored", 10, stored.size)

        // Verify each certificate is present
        certificateIds.forEach { certId ->
            assertEquals("cert-$certId", stored[certId]?.alias)
        }
    }

    @Test
    fun `rapid sequential certificate storage preserves all data`() = runTest {
        // Act: Store 5 certificates rapidly in sequence
        repeat(5) { index ->
            CertificateOrchestrator.markCertificateInstalled(context, index * 100, "cert-$index")
        }

        // Assert: All 5 should be stored
        val stored = getStoredCertificates()
        assertEquals(5, stored.size)

        repeat(5) { index ->
            assertEquals("cert-$index", stored[index * 100]?.alias)
        }
    }

    @Test
    fun `concurrent reads during writes see consistent data`() = runTest {
        // Arrange: Pre-populate with some certificates
        CertificateOrchestrator.markCertificateInstalled(context, 1, "cert-1")
        CertificateOrchestrator.markCertificateInstalled(context, 2, "cert-2")

        // Act: Concurrent write and read
        val writeJob = launch {
            CertificateOrchestrator.markCertificateInstalled(context, 3, "cert-3")
        }

        val readJob = launch {
            val certificates = CertificateOrchestrator.getCertificateStates(context)
            // Should see either 2 or 3 certificates (before or after write), but data should be consistent
            assertTrue(certificates.size >= 2)
        }

        writeJob.join()
        readJob.join()

        // Assert: Final state should have all 3
        val stored = getStoredCertificates()
        assertEquals(3, stored.size)
    }

    // ========== Test Category 4: Integration Tests ==========

    @Test
    fun `full enrollment flow stores certificate in DataStore after success`() = runTest {
        // Note: This test is limited because we can't easily mock ApiClient (it's an object)
        // Instead, we verify that if enrollment succeeds, DataStore storage happens

        // We'll test this by verifying the storeCertificateInstallation call happens
        // after a successful mock enrollment via the handler directly

        val template = createMockTemplate(123, "test-cert")

        // Create handler with mock client and installer
        val handler = CertificateEnrollmentHandler(
            scepClient = mockScepClient,
            certificateInstaller = mockInstaller,
        )

        // Act: Perform enrollment
        val result = handler.handleEnrollment(template)

        // Assert: Enrollment succeeded
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Success)

        // Manually verify the pattern - orchestrator would call storeCertificateInstallation
        val alias = (result as CertificateEnrollmentHandler.EnrollmentResult.Success).alias
        CertificateOrchestrator.markCertificateInstalled(context, 123, alias)

        // Verify it was stored
        val storedAlias = CertificateOrchestrator.getCertificateAlias(context, 123)
        assertNotNull(storedAlias)
        assertEquals(alias, storedAlias)
    }

    @Test
    fun `failed enrollment does not store in DataStore`() = runTest {
        // Arrange: Make SCEP enrollment fail
        mockScepClient.shouldThrowEnrollmentException = true

        val template = createMockTemplate(456, "failing-cert")

        val handler = CertificateEnrollmentHandler(
            scepClient = mockScepClient,
            certificateInstaller = mockInstaller,
        )

        // Act
        val result = handler.handleEnrollment(template)

        // Assert: Enrollment failed
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Failure)

        // Verify nothing was stored (orchestrator wouldn't call store on failure)
        val stored = getStoredCertificates()
        assertTrue(stored.isEmpty())
    }

    @Test
    fun `enrollment with custom installer uses provided installer`() = runTest {
        // This test verifies the dependency injection pattern works
        val customInstaller = MockCertificateInstaller()
        val template = createMockTemplate(789, "custom-cert")

        val handler = CertificateEnrollmentHandler(
            scepClient = mockScepClient,
            certificateInstaller = customInstaller,
        )

        // Act
        handler.handleEnrollment(template)

        // Assert: Custom installer was used
        assertTrue(customInstaller.wasInstallCalled)
        assertEquals("custom-cert", customInstaller.capturedAlias)

        // Original installer was not used
        assertFalse(mockInstaller.wasInstallCalled)
    }

    // ========== Test Category 4: Certificate Cleanup ==========

    // --- removeCertificateState tests ---

    @Test
    fun `removeCertificateState removes certificate from DataStore`() = runTest {
        // Arrange: Store 3 certificates
        storeTestCertificateInDataStore(1, "cert-1")
        storeTestCertificateInDataStore(2, "cert-2")
        storeTestCertificateInDataStore(3, "cert-3")

        // Act: Remove certificate 2
        CertificateOrchestrator.removeCertificateState(context, 2)

        // Assert: Only 1 and 3 remain
        val stored = getStoredCertificates()
        assertEquals(2, stored.size)
        assertTrue(stored.containsKey(1))
        assertFalse(stored.containsKey(2))
        assertTrue(stored.containsKey(3))
        assertEquals("cert-1", stored[1]?.alias)
        assertEquals("cert-3", stored[3]?.alias)
    }

    @Test
    fun `removeCertificateState handles non-existent certificate gracefully`() = runTest {
        // Arrange: Store 2 certificates
        storeTestCertificateInDataStore(1, "cert-1")
        storeTestCertificateInDataStore(2, "cert-2")

        // Act: Try to remove non-existent certificate
        CertificateOrchestrator.removeCertificateState(context, 999)

        // Assert: No exception thrown, DataStore unchanged
        val stored = getStoredCertificates()
        assertEquals(2, stored.size)
        assertTrue(stored.containsKey(1))
        assertTrue(stored.containsKey(2))
    }

    @Test
    fun `removeCertificateState handles corrupted DataStore gracefully`() = runTest {
        // Arrange: Corrupt DataStore with invalid JSON
        context.prefDataStore.edit { preferences ->
            preferences[stringPreferencesKey("installed_certificates")] = "{ invalid json }"
        }

        // Act: Try to remove certificate (should not throw)
        CertificateOrchestrator.removeCertificateState(context, 123)

        // Assert: No exception, operation succeeds
        // DataStore should be cleared/reset
        val stored = getStoredCertificates()
        assertTrue(stored.isEmpty())
    }

    @Test
    fun `concurrent removeCertificateState operations are thread-safe`() = runTest {
        // Arrange: Store 10 certificates
        repeat(10) { id ->
            storeTestCertificateInDataStore(id + 1, "cert-${id + 1}")
        }

        // Act: Remove certificates 2, 4, 6, 8, 10 in parallel
        val jobs = listOf(2, 4, 6, 8, 10).map { certId ->
            launch {
                CertificateOrchestrator.removeCertificateState(context, certId)
            }
        }
        jobs.forEach { it.join() }

        // Assert: Only odd-numbered certificates remain (1, 3, 5, 7, 9)
        val stored = getStoredCertificates()
        assertEquals(5, stored.size)
        listOf(1, 3, 5, 7, 9).forEach { id ->
            assertTrue("Certificate $id should exist", stored.containsKey(id))
        }
        listOf(2, 4, 6, 8, 10).forEach { id ->
            assertFalse("Certificate $id should not exist", stored.containsKey(id))
        }
    }

    @Test
    fun `cleanupRemovedCertificates handles removal flow correctly`() = runTest {
        data class StoredCert(val id: Int, val status: CertificateStatus = CertificateStatus.INSTALLED)
        data class ExpectedCert(val id: Int, val status: CertificateStatus?, val deleted: Boolean = false)
        data class TestCase(
            val name: String,
            val stored: List<StoredCert>,
            val hostCertificates: List<HostCertificate>,
            val expectedResultIds: Set<Int>,
            val expectedState: List<ExpectedCert>,
        )

        fun hostCert(id: Int, op: String) = HostCertificate(id, "verified", op)

        val testCases = listOf(
            TestCase(
                name = "marks INSTALLED cert as REMOVED when operation=remove",
                stored = listOf(StoredCert(1), StoredCert(2), StoredCert(3)),
                hostCertificates = listOf(hostCert(1, "install"), hostCert(2, "remove"), hostCert(3, "install")),
                expectedResultIds = setOf(2),
                expectedState = listOf(
                    ExpectedCert(1, CertificateStatus.INSTALLED),
                    ExpectedCert(2, CertificateStatus.REMOVED),
                    ExpectedCert(3, CertificateStatus.INSTALLED),
                ),
            ),
            TestCase(
                name = "skips already REMOVED cert",
                stored = listOf(StoredCert(1), StoredCert(2, CertificateStatus.REMOVED)),
                hostCertificates = listOf(hostCert(1, "install"), hostCert(2, "remove")),
                expectedResultIds = setOf(2),
                expectedState = listOf(
                    ExpectedCert(1, CertificateStatus.INSTALLED),
                    ExpectedCert(2, CertificateStatus.REMOVED),
                ),
            ),
            TestCase(
                name = "saves non-existent cert as REMOVED to prevent re-notification",
                stored = listOf(StoredCert(1)),
                hostCertificates = listOf(hostCert(1, "install"), hostCert(2, "remove")),
                expectedResultIds = setOf(2),
                expectedState = listOf(
                    ExpectedCert(1, CertificateStatus.INSTALLED),
                    ExpectedCert(2, CertificateStatus.REMOVED),
                ),
            ),
            TestCase(
                name = "deletes orphaned REMOVED cert from DataStore",
                stored = listOf(StoredCert(1), StoredCert(2), StoredCert(3, CertificateStatus.REMOVED)),
                hostCertificates = listOf(hostCert(1, "install"), hostCert(2, "install")),
                expectedResultIds = setOf(3),
                expectedState = listOf(
                    ExpectedCert(1, CertificateStatus.INSTALLED),
                    ExpectedCert(2, CertificateStatus.INSTALLED),
                    ExpectedCert(3, null, deleted = true),
                ),
            ),
            TestCase(
                name = "marks orphaned INSTALLED cert as REMOVED",
                stored = listOf(StoredCert(1), StoredCert(2), StoredCert(3)),
                hostCertificates = listOf(hostCert(1, "install"), hostCert(2, "install")),
                expectedResultIds = setOf(3),
                expectedState = listOf(
                    ExpectedCert(1, CertificateStatus.INSTALLED),
                    ExpectedCert(2, CertificateStatus.INSTALLED),
                    ExpectedCert(3, CertificateStatus.REMOVED),
                ),
            ),
            TestCase(
                name = "returns empty when all are install operations",
                stored = listOf(StoredCert(1), StoredCert(2)),
                hostCertificates = listOf(hostCert(1, "install"), hostCert(2, "install")),
                expectedResultIds = emptySet(),
                expectedState = listOf(
                    ExpectedCert(1, CertificateStatus.INSTALLED),
                    ExpectedCert(2, CertificateStatus.INSTALLED),
                ),
            ),
        )

        for (case in testCases) {
            clearDataStore()
            case.stored.forEach { storeTestCertificateInDataStore(it.id, "cert-${it.id}", it.status) }

            val results = CertificateOrchestrator.cleanupRemovedCertificates(context, case.hostCertificates)
            val stored = getStoredCertificates()

            assertEquals("Result IDs - ${case.name}", case.expectedResultIds, results.keys)
            for (expected in case.expectedState) {
                if (expected.deleted) {
                    assertFalse("Cert ${expected.id} should be deleted - ${case.name}", stored.containsKey(expected.id))
                } else {
                    assertEquals("Cert ${expected.id} status - ${case.name}", expected.status, stored[expected.id]?.status)
                }
            }
        }
    }

    // ========== Helper Methods for Tests ==========

    private fun createMockTemplate(
        id: Int,
        name: String,
        url: String = "https://scep.example.com/scep",
        challenge: String = "test-challenge",
    ): GetCertificateTemplateResponse = GetCertificateTemplateResponse(
        id = id,
        name = name,
        certificateAuthorityId = 123,
        certificateAuthorityName = "Test CA",
        createdAt = "2024-01-01T00:00:00Z",
        subjectName = "CN=$name,O=FleetDM",
        certificateAuthorityType = "SCEP",
        status = "active",
        scepChallenge = challenge,
        fleetChallenge = "fleet-secret",
        keyLength = 2048,
        signatureAlgorithm = "SHA256withRSA",
        url = url,
    )
}
