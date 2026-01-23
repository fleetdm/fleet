package com.fleetdm.agent.scep

import java.math.BigInteger
import java.security.PrivateKey
import java.security.cert.Certificate
import java.util.Date

/**
 * Result of a successful SCEP enrollment containing the private key and certificate chain.
 *
 * @property privateKey The generated private key
 * @property certificateChain The certificate chain from the SCEP server (leaf certificate first)
 * @property notAfter The expiration date (notAfter) of the leaf certificate
 * @property notBefore The effective date (notBefore) of the leaf certificate
 * @property serialNumber The serial number of the leaf certificate
 */
data class ScepResult(
    val privateKey: PrivateKey,
    val certificateChain: List<Certificate>,
    val notAfter: Date,
    val notBefore: Date,
    val serialNumber: BigInteger,
)
