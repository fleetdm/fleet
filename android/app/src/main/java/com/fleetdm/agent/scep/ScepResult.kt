package com.fleetdm.agent.scep

import java.security.PrivateKey
import java.security.cert.Certificate

/**
 * Result of a successful SCEP enrollment containing the private key and certificate chain.
 *
 * @property privateKey The generated private key
 * @property certificateChain The certificate chain from the SCEP server (leaf certificate first)
 */
data class ScepResult(val privateKey: PrivateKey, val certificateChain: Array<Certificate>) {
    override fun equals(other: Any?): Boolean {
        if (this === other) return true
        if (javaClass != other?.javaClass) return false

        other as ScepResult

        if (privateKey != other.privateKey) return false
        if (!certificateChain.contentEquals(other.certificateChain)) return false

        return true
    }

    override fun hashCode(): Int {
        var result = privateKey.hashCode()
        result = 31 * result + certificateChain.contentHashCode()
        return result
    }
}
