package com.fleetdm.agent.testutil

import com.fleetdm.agent.CertificateApiClient
import com.fleetdm.agent.CertificateTemplateResult
import com.fleetdm.agent.DeviceCACertificateResponse
import com.fleetdm.agent.UpdateCertificateStatusOperation
import com.fleetdm.agent.UpdateCertificateStatusStatus
import java.math.BigInteger
import java.util.Date

/**
 * Represents a captured call to updateCertificateStatus for test assertions.
 */
data class UpdateStatusCall(
    val certificateId: Int,
    val status: UpdateCertificateStatusStatus,
    val operationType: UpdateCertificateStatusOperation,
    val detail: String?,
    val notAfter: Date?,
    val notBefore: Date?,
    val serialNumber: BigInteger?,
)

/**
 * Fake implementation of CertificateApiClient for testing.
 * Provides configurable handlers and captures calls for assertions.
 */
class FakeCertificateApiClient : CertificateApiClient {
    var getCertificateTemplateHandler: (Int) -> Result<CertificateTemplateResult> = {
        Result.failure(Exception("getCertificateTemplate not configured"))
    }
    var updateCertificateStatusHandler: (UpdateStatusCall) -> Result<Unit> = { Result.success(Unit) }

    private val _updateStatusCalls = mutableListOf<UpdateStatusCall>()
    val updateStatusCalls: List<UpdateStatusCall> get() = _updateStatusCalls.toList()

    private val _getCertificateTemplateCalls = mutableListOf<Int>()
    val getCertificateTemplateCalls: List<Int> get() = _getCertificateTemplateCalls.toList()

    override suspend fun getCertificateTemplate(certificateId: Int): Result<CertificateTemplateResult> {
        _getCertificateTemplateCalls.add(certificateId)
        return getCertificateTemplateHandler(certificateId)
    }

    override suspend fun updateCertificateStatus(
        certificateId: Int,
        status: UpdateCertificateStatusStatus,
        operationType: UpdateCertificateStatusOperation,
        detail: String?,
        notAfter: Date?,
        notBefore: Date?,
        serialNumber: BigInteger?,
    ): Result<Unit> {
        val call = UpdateStatusCall(
            certificateId,
            status,
            operationType,
            detail,
            notAfter,
            notBefore,
            serialNumber,
        )
        _updateStatusCalls.add(call)
        return updateCertificateStatusHandler(call)
    }

    var getCaCertificateHandler: (Int) -> Result<DeviceCACertificateResponse> = {
        Result.failure(Exception("getCaCertificate not configured"))
    }
    var updateCaCertificateStatusHandler: (
        Int,
        UpdateCertificateStatusStatus,
        String?,
    ) -> Result<Unit> = { _, _, _ -> Result.success(Unit) }

    private val _getCaCertificateCalls = mutableListOf<Int>()
    val getCaCertificateCalls: List<Int> get() = _getCaCertificateCalls.toList()

    private val _updateCaStatusCalls = mutableListOf<Triple<Int, UpdateCertificateStatusStatus, String?>>()
    val updateCaStatusCalls: List<Triple<Int, UpdateCertificateStatusStatus, String?>> get() = _updateCaStatusCalls.toList()

    override suspend fun getCaCertificate(caId: Int): Result<DeviceCACertificateResponse> {
        _getCaCertificateCalls.add(caId)
        return getCaCertificateHandler(caId)
    }

    override suspend fun updateCaCertificateStatus(caId: Int, status: UpdateCertificateStatusStatus, detail: String?): Result<Unit> {
        _updateCaStatusCalls.add(Triple(caId, status, detail))
        return updateCaCertificateStatusHandler(caId, status, detail)
    }

    fun reset() {
        getCertificateTemplateHandler = { Result.failure(Exception("getCertificateTemplate not configured")) }
        updateCertificateStatusHandler = { Result.success(Unit) }
        getCaCertificateHandler = { Result.failure(Exception("getCaCertificate not configured")) }
        updateCaCertificateStatusHandler = { _, _, _ -> Result.success(Unit) }
        _updateStatusCalls.clear()
        _getCertificateTemplateCalls.clear()
        _getCaCertificateCalls.clear()
        _updateCaStatusCalls.clear()
    }
}
