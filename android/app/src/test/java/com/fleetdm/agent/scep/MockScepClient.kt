package com.fleetdm.agent.scep

import com.fleetdm.agent.GetCertificateTemplateResponse
import org.bouncycastle.asn1.x500.X500Name
import org.bouncycastle.cert.jcajce.JcaX509CertificateConverter
import org.bouncycastle.cert.jcajce.JcaX509v3CertificateBuilder
import org.bouncycastle.jce.provider.BouncyCastleProvider
import org.bouncycastle.operator.jcajce.JcaContentSignerBuilder
import java.math.BigInteger
import java.security.KeyPair
import java.security.KeyPairGenerator
import java.security.Security
import java.security.cert.Certificate
import java.security.cert.X509Certificate
import java.util.Date

/**
 * Mock implementation of ScepClient for testing.
 * Provides configurable behavior for different test scenarios.
 */
class MockScepClient : ScepClient {

    var shouldSucceed = true
    var shouldThrowEnrollmentException = false
    var shouldThrowNetworkException = false
    var shouldThrowCertificateException = false
    var enrollmentDelay = 0L
    var capturedConfig: GetCertificateTemplateResponse? = null
    var capturedScepUrl: String? = null

    init {
        if (Security.getProvider(BouncyCastleProvider.PROVIDER_NAME) == null) {
            Security.addProvider(BouncyCastleProvider())
        }
    }

    override suspend fun enroll(config: GetCertificateTemplateResponse, scepUrl: String): ScepResult {
        capturedConfig = config
        capturedScepUrl = scepUrl

        if (enrollmentDelay > 0) {
            kotlinx.coroutines.delay(enrollmentDelay)
        }

        when {
            shouldThrowNetworkException -> throw ScepNetworkException("Mock network error")
            shouldThrowEnrollmentException -> throw ScepEnrollmentException("Mock enrollment failed")
            shouldThrowCertificateException -> throw ScepCertificateException("Mock certificate error")
            !shouldSucceed -> throw ScepException("Mock general error")
        }

        // Generate a real key pair and certificate for testing
        return generateMockResult(config.subjectName)
    }

    private fun generateMockResult(subject: String): ScepResult {
        val keyPairGen = KeyPairGenerator.getInstance("RSA")
        keyPairGen.initialize(2048)
        val keyPair = keyPairGen.genKeyPair()

        val cert = generateSelfSignedCertificate(keyPair, subject)

        // Extract certificate metadata from generated certificate
        val x509Cert = cert as X509Certificate
        val notAfter = x509Cert.notAfter
        val notBefore = x509Cert.notBefore
        val serialNumber = x509Cert.serialNumber

        return ScepResult(
            privateKey = keyPair.private,
            certificateChain = listOf(cert),
            notAfter = notAfter,
            notBefore = notBefore,
            serialNumber = serialNumber,
        )
    }

    private fun generateSelfSignedCertificate(keyPair: KeyPair, subject: String): Certificate {
        val now = System.currentTimeMillis()
        val validityEnd = now + (1000L * 60 * 60 * 24 * 365)

        val entity = X500Name(subject)

        val certBuilder = JcaX509v3CertificateBuilder(
            entity,
            BigInteger.valueOf(System.currentTimeMillis()),
            Date(now),
            Date(validityEnd),
            entity,
            keyPair.public,
        )

        val contentSigner = JcaContentSignerBuilder("SHA256withRSA").build(keyPair.private)
        val certHolder = certBuilder.build(contentSigner)

        return JcaX509CertificateConverter().getCertificate(certHolder)
    }

    fun reset() {
        shouldSucceed = true
        shouldThrowEnrollmentException = false
        shouldThrowNetworkException = false
        shouldThrowCertificateException = false
        enrollmentDelay = 0L
        capturedConfig = null
        capturedScepUrl = null
    }
}
