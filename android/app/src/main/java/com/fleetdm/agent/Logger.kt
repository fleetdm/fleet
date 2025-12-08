package com.fleetdm.agent

/**
 * Interface for logging - allows different implementations
 * (production uses Android Log, tests use mocks or no-op).
 */
interface Logger {
    fun w(tag: String, message: String, throwable: Throwable? = null)
    fun e(tag: String, message: String, throwable: Throwable? = null)
}

/**
 * No-op logger for testing.
 */
class NoOpLogger : Logger {
    override fun w(tag: String, message: String, throwable: Throwable?) = Unit
    override fun e(tag: String, message: String, throwable: Throwable?) = Unit
}
