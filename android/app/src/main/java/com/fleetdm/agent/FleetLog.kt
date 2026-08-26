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

    // logcat drops whatever a single entry carries past ~4 KB of payload, measured in bytes.
    private const val MAX_LOGCAT_CHUNK_BYTES = 3000

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

    private fun logChunked(tag: String, text: String) {
        chunkForLogcat(text, MAX_LOGCAT_CHUNK_BYTES).forEach { Log.e(tag, it) }
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

/**
 * Splits [text] into pieces of at most [maxBytes] UTF-8 bytes, breaking at line boundaries where it
 * can. Log.e with a throwable splits a long trace across entries for us; pre-rendered text is
 * truncated instead, and the limit logcat applies is a byte count, so measuring characters would
 * still lose the tail of non-ASCII text.
 */
internal fun chunkForLogcat(text: String, maxBytes: Int): List<String> {
    if (text.utf8Size() <= maxBytes) {
        return listOf(text)
    }

    val chunks = mutableListOf<String>()
    var current = StringBuilder()
    var currentBytes = 0
    // Tracked separately from the builder's length: an empty line is a piece too, and dropping it
    // because it added no characters would silently delete blank lines from the text.
    var started = false
    for (piece in text.lineSequence().flatMap { splitByUtf8Bytes(it, maxBytes) }) {
        val pieceBytes = piece.utf8Size()
        if (started && currentBytes + 1 + pieceBytes > maxBytes) {
            chunks.add(current.toString())
            current = StringBuilder()
            currentBytes = 0
            started = false
        }
        if (started) {
            current.append('\n')
            currentBytes++
        }
        current.append(piece)
        currentBytes += pieceBytes
        started = true
    }
    if (started) {
        chunks.add(current.toString())
    }
    return chunks
}

/** Splits one line into pieces of at most [maxBytes] UTF-8 bytes, never mid-character. */
private fun splitByUtf8Bytes(line: String, maxBytes: Int): Sequence<String> {
    if (line.utf8Size() <= maxBytes) {
        return sequenceOf(line)
    }
    return sequence {
        val piece = StringBuilder()
        var pieceBytes = 0
        var index = 0
        while (index < line.length) {
            val codePoint = line.codePointAt(index)
            val width = Character.charCount(codePoint)
            val size = utf8Size(codePoint)
            if (pieceBytes + size > maxBytes) {
                yield(piece.toString())
                piece.setLength(0)
                pieceBytes = 0
            }
            piece.append(line, index, index + width)
            pieceBytes += size
            index += width
        }
        if (piece.isNotEmpty()) {
            yield(piece.toString())
        }
    }
}

private fun String.utf8Size(): Int = toByteArray(Charsets.UTF_8).size

private fun utf8Size(codePoint: Int): Int = when {
    codePoint < 0x80 -> 1
    codePoint < 0x800 -> 2
    codePoint < 0x10000 -> 3
    else -> 4
}
