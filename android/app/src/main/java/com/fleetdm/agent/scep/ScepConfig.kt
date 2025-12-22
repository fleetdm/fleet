package com.fleetdm.agent.scep

/**
 * Configuration for SCEP enrollment.
 *
 * @property url The SCEP server enrollment URL
 * @property challenge The challenge password for authentication
 * @property alias The certificate alias for silent installation
 * @property subject The X.500 distinguished name subject (e.g., "CN=Device123,O=FleetDM")
 * @property keyLength RSA key length in bits (default: 2048)
 * @property signatureAlgorithm Signature algorithm to use (default: SHA256withRSA)
 */
data class ScepConfig(
    val url: String,
    val challenge: String,
    val alias: String,
    val subject: String,
    val keyLength: Int = 2048,
    val signatureAlgorithm: String = "SHA256withRSA",
) {
    init {
        require(url.isNotBlank()) { "SCEP URL cannot be blank" }
        require(url.startsWith("http://") || url.startsWith("https://")) {
            "SCEP URL must start with http:// or https://"
        }
        require(challenge.isNotBlank()) { "Challenge password cannot be blank" }
        require(alias.isNotBlank()) { "Certificate alias cannot be blank" }
        require(subject.isNotBlank()) { "Subject cannot be blank" }
        require(keyLength >= 2048) { "Key length must be at least 2048 bits" }
        require(signatureAlgorithm.isNotBlank()) { "Signature algorithm cannot be blank" }
    }
}
