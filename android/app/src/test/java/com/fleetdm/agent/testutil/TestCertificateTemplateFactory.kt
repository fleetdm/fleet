package com.fleetdm.agent.testutil

import com.fleetdm.agent.GetCertificateTemplateResponse

/**
 * Factory for creating GetCertificateTemplateResponse instances in tests.
 * Provides sensible defaults for all fields to reduce boilerplate.
 */
object TestCertificateTemplateFactory {

    fun create(
        id: Int = 1,
        name: String = "test-cert",
        certificateAuthorityId: Int = 123,
        certificateAuthorityName: String = "Test CA",
        createdAt: String = "2024-01-01T00:00:00Z",
        subjectName: String = "CN=Test,O=FleetDM",
        certificateAuthorityType: String = "SCEP",
        status: String = "active",
        scepChallenge: String = "test-challenge",
        fleetChallenge: String = "fleet-secret",
        keyLength: Int = 2048,
        signatureAlgorithm: String = "SHA256withRSA",
        url: String = "https://scep.example.com/cgi-bin/pkiclient.exe",
    ): GetCertificateTemplateResponse = GetCertificateTemplateResponse(
        id = id,
        name = name,
        certificateAuthorityId = certificateAuthorityId,
        certificateAuthorityName = certificateAuthorityName,
        createdAt = createdAt,
        subjectName = subjectName,
        certificateAuthorityType = certificateAuthorityType,
        status = status,
        scepChallenge = scepChallenge,
        fleetChallenge = fleetChallenge,
        keyLength = keyLength,
        signatureAlgorithm = signatureAlgorithm,
        url = url,
    )
}
