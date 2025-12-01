package com.fleetdm.agent.scep

/**
 * Base exception for all SCEP-related errors.
 */
open class ScepException(message: String, cause: Throwable? = null) : Exception(message, cause)

/**
 * Thrown when SCEP enrollment fails due to server rejection or pending status.
 */
class ScepEnrollmentException(message: String, cause: Throwable? = null) : ScepException(message, cause)

/**
 * Thrown when network communication with the SCEP server fails.
 */
class ScepNetworkException(message: String, cause: Throwable? = null) : ScepException(message, cause)

/**
 * Thrown when certificate processing or validation fails.
 */
class ScepCertificateException(message: String, cause: Throwable? = null) : ScepException(message, cause)

/**
 * Thrown when key pair generation fails.
 */
class ScepKeyGenerationException(message: String, cause: Throwable? = null) : ScepException(message, cause)

/**
 * Thrown when CSR (Certificate Signing Request) creation fails.
 */
class ScepCsrException(message: String, cause: Throwable? = null) : ScepException(message, cause)
