package com.fleetdm.agent

import android.content.Context
import android.util.Log
import java.io.File
import java.nio.file.Files
import java.nio.file.StandardCopyOption
import java.time.Instant

/**
 * Persistent error logger that writes to internal storage alongside the normal logcat output.
 *
 * Call [initialize] once in Application.onCreate(). Then use [e] wherever you would call
 * Log.e — it writes to logcat AND appends to fleet_errors.log in filesDir. Messages and stack
 * traces are passed through [redactSecrets] first, so enrollment secrets stay out of both.
 * The log file is capped at 512 KB; when it exceeds that size the current file is renamed
 * to fleet_errors.log.1 and a fresh file is started.
 */
object FleetLog {
    private const val TAG = "FleetLog"
    private const val LOG_FILE_NAME = "fleet_errors.log"
    private const val MAX_SIZE_BYTES = 512 * 1024L // 512 KB

    // logcat drops whatever a single entry carries past ~4 KB of payload.
    private const val MAX_LOGCAT_CHUNK_CHARS = 3000

    @Volatile private var logFile: File? = null

    fun initialize(context: Context) {
        logFile = File(context.filesDir, LOG_FILE_NAME)
    }

    fun e(tag: String, msg: String, throwable: Throwable? = null) {
        // Render the throwable ourselves rather than handing it to Log.e: Log.e prints the whole
        // cause chain, and causes raised by the HTTP stack can carry the SCEP proxy URL's challenge.
        val (message, stackTrace) = renderLogEntry(msg, throwable)
        logChunked(tag, if (stackTrace == null) message else "$message\n$stackTrace")
        appendToFile(tag, message, stackTrace)
    }

    /**
     * Writes [text] to logcat, split across entries so a long stack trace isn't cut off. Log.e with
     * a throwable splits for us; passing pre-rendered text doesn't, it truncates.
     */
    private fun logChunked(tag: String, text: String) {
        if (text.length <= MAX_LOGCAT_CHUNK_CHARS) {
            Log.e(tag, text)
            return
        }
        val pieces = text.lineSequence().flatMap { line ->
            if (line.length <= MAX_LOGCAT_CHUNK_CHARS) sequenceOf(line) else line.chunked(MAX_LOGCAT_CHUNK_CHARS).asSequence()
        }
        val chunk = StringBuilder()
        pieces.forEach { piece ->
            if (chunk.isNotEmpty() && chunk.length + piece.length + 1 > MAX_LOGCAT_CHUNK_CHARS) {
                Log.e(tag, chunk.toString())
                chunk.setLength(0)
            }
            if (chunk.isNotEmpty()) {
                chunk.append('\n')
            }
            chunk.append(piece)
        }
        if (chunk.isNotEmpty()) {
            Log.e(tag, chunk.toString())
        }
    }

    // Redacted on the way out too: a file written by an older build may still hold secrets.
    fun readErrors(): String = (logFile?.takeIf { it.exists() }?.readText() ?: "").redactSecrets()

    @Synchronized
    private fun appendToFile(tag: String, msg: String, stackTrace: String?) {
        val file = logFile ?: return
        try {
            if (file.exists() && file.length() > MAX_SIZE_BYTES) {
                val backup = File(file.parent, "$LOG_FILE_NAME.1")
                backup.delete()
                if (!file.renameTo(backup)) {
                    Log.w(TAG, "Failed to rotate log file; continuing to append to existing file")
                }
            }

            val timestamp = Instant.now().toString()
            val sb = StringBuilder()
            sb.append("$timestamp E $tag $msg\n")
            stackTrace?.let { sb.append(it.prependIndent("    ")).append("\n") }
            file.appendText(sb.toString())
        } catch (e: Exception) {
            Log.e(TAG, "Failed to write to log file: ${e.message}")
        }
    }
}
