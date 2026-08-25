package com.fleetdm.agent

/** Stands in for a secret that must not reach logs. */
const val REDACTED = "<redacted>"

// Fleet SCEP proxy URLs carry a one-time challenge as the last field of the comma-separated path
// identifier ("{hostUUID},g{templateID},{caType},{challenge}"). jScep appends a query string to it,
// so the identifier ends at the first "?" or whitespace.
private val SCEP_PROXY_IDENTIFIER = Regex("""(/mdm/scep/proxy/)([^/?\s]+)""")

// Commas are percent-encoded in some Fleet-generated SCEP URLs.
private val IDENTIFIER_FIELD_SEPARATOR = Regex(""",|%2C""", RegexOption.IGNORE_CASE)

// hostUUID, template ID and CA type are safe to log; everything after them is the challenge.
private const val NON_SECRET_IDENTIFIER_FIELDS = 3

/**
 * Strips enrollment secrets from text on its way to a log, log file, or status detail.
 *
 * Redacts the challenge in Fleet SCEP proxy URLs, which reach us inside exception messages thrown
 * by the HTTP stack and jScep. Anyone with a USB cable can read logcat, so a challenge that lands
 * there is a challenge anyone can use.
 */
fun String.redactSecrets(): String = SCEP_PROXY_IDENTIFIER.replace(this) { match ->
    val (prefix, identifier) = match.destructured
    val separators = IDENTIFIER_FIELD_SEPARATOR.findAll(identifier).take(NON_SECRET_IDENTIFIER_FIELDS).toList()
    val challengeStart = separators.lastOrNull()?.range?.last?.plus(1) ?: identifier.length
    if (separators.size < NON_SECRET_IDENTIFIER_FIELDS || challengeStart == identifier.length) {
        // No challenge field, or it is empty: nothing to hide.
        match.value
    } else {
        prefix + identifier.take(challengeStart) + REDACTED
    }
}

/**
 * Renders a nullable secret for a log message: whether it is set stays visible, its value does not.
 */
fun String?.redacted(): String = when {
    this == null -> "null"
    isEmpty() -> "\"\""
    else -> REDACTED
}
