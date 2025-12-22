package com.fleetdm.agent.scep

import java.security.PrivateKey
import java.security.cert.Certificate

/**
 * Result of a successful SCEP enrollment containing the private key and certificate chain.
 *
 * @property privateKey The generated private key
 * @property certificateChain The certificate chain from the SCEP server (leaf certificate first)
 */
data class ScepResult(val privateKey: PrivateKey, val certificateChain: List<Certificate>)
