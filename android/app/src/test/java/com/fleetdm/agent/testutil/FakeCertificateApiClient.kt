package com.fleetdm.agent.testutil

import com.fleetdm.agent.CertificateApiClient
import com.fleetdm.agent.CertificateTemplateResult
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

    fun reset() {
        getCertificateTemplateHandler = { Result.failure(Exception("getCertificateTemplate not configured")) }
        updateCertificateStatusHandler = { Result.success(Unit) }
        _updateStatusCalls.clear()
        _getCertificateTemplateCalls.clear()
    }
}
