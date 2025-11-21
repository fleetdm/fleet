package com.fleetdm.agent.scep

import android.util.Log
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.bouncycastle.asn1.DERPrintableString
import org.bouncycastle.asn1.pkcs.PKCSObjectIdentifiers
import org.bouncycastle.asn1.x500.X500Name
import org.bouncycastle.cert.jcajce.JcaX509CertificateConverter
import org.bouncycastle.cert.jcajce.JcaX509v3CertificateBuilder
import org.bouncycastle.jce.provider.BouncyCastleProvider
import org.bouncycastle.operator.jcajce.JcaContentSignerBuilder
import org.bouncycastle.pkcs.jcajce.JcaPKCS10CertificationRequestBuilder
import org.jscep.client.Client
import org.jscep.client.verification.OptimisticCertificateVerifier
import java.math.BigInteger
import java.net.URL
import java.security.KeyPairGenerator
import java.security.Security
import java.security.cert.Certificate
import java.util.Date

/**
 * Implementation of ScepClient using jScep library and BouncyCastle cryptography.
 *
 * This implementation performs SCEP enrollment by:
 * 1. Generating an RSA key pair
 * 2. Creating a self-signed certificate for PKCS7 envelope signing
 * 3. Building a Certificate Signing Request (CSR) with challenge password
 * 4. Sending enrollment request to SCEP server
 * 5. Extracting and returning the issued certificate and private key
 */
class ScepClientImpl : ScepClient {

    companion object {
        private const val TAG = "ScepClientImpl"
        private const val SCEP_PROFILE = "NDESCA" // Network Device Enrollment Service CA
        private const val SELF_SIGNED_CERT_VALIDITY_DAYS = 100L

        init {
            // Ensure BouncyCastle provider is loaded
            if (Security.getProvider(BouncyCastleProvider.PROVIDER_NAME) == null) {
                Security.addProvider(BouncyCastleProvider())
                Log.d(TAG, "BouncyCastle security provider loaded")
            }
        }
    }

    override suspend fun enroll(config: ScepConfig): ScepResult = withContext(Dispatchers.IO) {
        try {
            Log.i(TAG, "Starting SCEP enrollment")
            Log.d(TAG, "SCEP URL: ${config.url}")
            Log.d(TAG, "Subject: ${config.subject}")
            Log.d(TAG, "Key length: ${config.keyLength} bits")

            // Step 1: Generate key pair
            val keyPair = generateKeyPair(config.keyLength)
            Log.i(TAG, "Key pair generated successfully")

            // Step 2: Parse subject name
            val entity = try {
                X500Name(config.subject)
            } catch (e: Exception) {
                throw ScepCsrException("Invalid X.500 subject name: ${config.subject}", e)
            }

            // Step 3: Create self-signed certificate for signing the PKCS7 envelope
            val selfSignedCert = createSelfSignedCertificate(
                entity,
                keyPair,
                config.signatureAlgorithm
            )
            Log.i(TAG, "Self-signed certificate created")

            // Step 4: Create SCEP client
            val server = try {
                URL(config.url)
            } catch (e: Exception) {
                throw ScepNetworkException("Invalid SCEP URL: ${config.url}", e)
            }

            val verifier = OptimisticCertificateVerifier()
            val client = Client(server, verifier)
            Log.d(TAG, "SCEP client initialized")

            // Step 5: Build Certificate Signing Request (CSR)
            val csr = buildCsr(entity, keyPair, config.challenge, config.signatureAlgorithm)
            Log.i(TAG, "Certificate Signing Request created")

            // Step 6: Send enrollment request
            Log.i(TAG, "Sending enrollment request to SCEP server...")
            val response = try {
                client.enrol(selfSignedCert, keyPair.private, csr, SCEP_PROFILE)
            } catch (e: Exception) {
                throw ScepNetworkException("Failed to communicate with SCEP server", e)
            }

            // Step 7: Process response
            when {
                response.isSuccess -> {
                    Log.i(TAG, "Enrollment successful!")
                    val certificates = extractCertificates(response.certStore)

                    if (certificates.isEmpty()) {
                        throw ScepCertificateException("No certificates returned from SCEP server")
                    }

                    Log.d(TAG, "Received ${certificates.size} certificate(s) from server")

                    ScepResult(
                        privateKey = keyPair.private,
                        certificateChain = certificates.toTypedArray()
                    )
                }
                response.isPending -> {
                    throw ScepEnrollmentException(
                        "Enrollment is pending - requires CA administrator approval"
                    )
                }
                else -> {
                    throw ScepEnrollmentException(
                        "Enrollment failed - certificate not issued by SCEP server"
                    )
                }
            }
        } catch (e: ScepException) {
            Log.e(TAG, "SCEP enrollment failed: ${e.message}", e)
            throw e
        } catch (e: Exception) {
            Log.e(TAG, "Unexpected error during SCEP enrollment", e)
            throw ScepException("Unexpected SCEP enrollment error: ${e.message}", e)
        }
    }

    private fun generateKeyPair(keyLength: Int) = try {
        val keyGen = KeyPairGenerator.getInstance("RSA")
        keyGen.initialize(keyLength)
        keyGen.genKeyPair()
    } catch (e: Exception) {
        throw ScepKeyGenerationException("Failed to generate RSA key pair", e)
    }

    private fun createSelfSignedCertificate(
        entity: X500Name,
        keyPair: java.security.KeyPair,
        signatureAlgorithm: String
    ) = try {
        val now = System.currentTimeMillis()
        val validityEnd = now + (1000L * 60 * 60 * 24 * SELF_SIGNED_CERT_VALIDITY_DAYS)

        val certBuilder = JcaX509v3CertificateBuilder(
            entity,
            BigInteger.valueOf(1),
            Date(now),
            Date(validityEnd),
            entity,
            keyPair.public
        )

        val contentSigner = JcaContentSignerBuilder(signatureAlgorithm).build(keyPair.private)
        val certHolder = certBuilder.build(contentSigner)

        JcaX509CertificateConverter().getCertificate(certHolder)
    } catch (e: Exception) {
        throw ScepCertificateException("Failed to create self-signed certificate", e)
    }

    private fun buildCsr(
        entity: X500Name,
        keyPair: java.security.KeyPair,
        challenge: String,
        signatureAlgorithm: String
    ) = try {
        val csrBuilder = JcaPKCS10CertificationRequestBuilder(entity, keyPair.public)

        // Add challenge password attribute
        val passwordAttr = DERPrintableString(challenge)
        csrBuilder.addAttribute(PKCSObjectIdentifiers.pkcs_9_at_challengePassword, passwordAttr)

        val contentSigner = JcaContentSignerBuilder(signatureAlgorithm).build(keyPair.private)
        csrBuilder.build(contentSigner)
    } catch (e: Exception) {
        throw ScepCsrException("Failed to build Certificate Signing Request", e)
    }

    private fun extractCertificates(certStore: java.security.cert.CertStore): List<Certificate> = try {
        val certificates = certStore.getCertificates(null)
        certificates.toList()
    } catch (e: Exception) {
        throw ScepCertificateException("Failed to extract certificates from response", e)
    }
}
