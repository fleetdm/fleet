package com.fleetdm.agent

/** Stands in for a secret that must not reach logs. */
internal const val REDACTED = "<redacted>"

/** Path of Fleet's SCEP proxy. Its URLs end with a one-time challenge. */
internal const val SCEP_PROXY_PATH = "/mdm/scep/proxy/"

// A SCEP proxy URL ends with the comma-separated identifier
// "{hostUUID},g{templateID},{caType},{challenge}". jScep appends a query string, and whatever
// message carries the URL often quotes or brackets it, so the identifier stops at the first "?",
// whitespace, or closing delimiter. Dots stay allowed: a host UUID may contain one, and stopping
// short of it would leave the challenge in the text.
private val SCEP_PROXY_IDENTIFIER = Regex("""(${Regex.escape(SCEP_PROXY_PATH)})([^/?\s"'<>)\]}]+)""")

// Commas are percent-encoded in some Fleet-generated SCEP URLs.
private val IDENTIFIER_FIELD_SEPARATOR = Regex(""",|%2C""", RegexOption.IGNORE_CASE)

// hostUUID, template ID and CA type are safe to log; everything after them is the challenge.
private const val NON_SECRET_IDENTIFIER_FIELDS = 3

// Challenges as they appear in a certificate template JSON body. kotlinx.serialization quotes the
// offending input in its decoding errors, so a body that fails to parse can reach a log line. The
// value is matched as a JSON string, escapes included, so a challenge holding a quote doesn't cut
// the match short. The closing quote is optional because the quoted input may be cut off mid-value.
private val JSON_CHALLENGE_FIELD =
    Regex("\"(scep_challenge|fleet_challenge)\"\\s*:\\s*\"[^\"\\\\]*(?:\\\\.[^\"\\\\]*)*\\\\?(\"|$)")

/**
 * Strips enrollment secrets from text on its way to a log, log file, or status detail.
 *
 * Redacts the challenge in Fleet SCEP proxy URLs and in certificate template JSON, both of which
 * reach us inside exception messages thrown by the HTTP and serialization stacks. Anyone with a USB
 * cable can read logcat, so a challenge that lands there is a challenge anyone can use.
 */
internal fun String.redactSecrets(): String = redactScepProxyUrls().redactJsonChallenges()

private fun String.redactScepProxyUrls(): String = SCEP_PROXY_IDENTIFIER.replace(this) { match ->
    val (prefix, identifier) = match.destructured
    val separators = IDENTIFIER_FIELD_SEPARATOR.findAll(identifier).take(NON_SECRET_IDENTIFIER_FIELDS).toList()
    val challengeStart = separators.lastOrNull()?.range?.last?.plus(1) ?: identifier.length
    if (separators.size < NON_SECRET_IDENTIFIER_FIELDS || challengeStart == identifier.length) {
        // Identifiers with fewer fields carry no challenge, and neither does an empty last field.
        match.value
    } else {
        prefix + identifier.take(challengeStart) + REDACTED
    }
}

private fun String.redactJsonChallenges(): String = JSON_CHALLENGE_FIELD.replace(this) { match ->
    "\"${match.groupValues[1]}\":\"$REDACTED\""
}

/**
 * Renders a nullable secret for a log message: whether it is set stays visible, its value does not.
 */
internal fun String?.redacted(): String = when {
    this == null -> "null"
    isEmpty() -> "\"\""
    else -> REDACTED
}

/**
 * Builds the redacted text [FleetLog] writes for one entry: the message, and the stack trace of the
 * throwable (cause chain included) when there is one.
 */
internal fun renderLogEntry(msg: String, throwable: Throwable?): Pair<String, String?> =
    msg.redactSecrets() to throwable?.stackTraceToString()?.trimEnd()?.redactSecrets()
