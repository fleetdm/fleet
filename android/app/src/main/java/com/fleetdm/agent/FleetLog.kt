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
 * Log.e — it writes to logcat AND appends to fleet_errors.log in filesDir.
 * The log file is capped at 512 KB; when it exceeds that size the current file is renamed
 * to fleet_errors.log.1 and a fresh file is started.
 */
object FleetLog {
    private const val TAG = "FleetLog"
    private const val LOG_FILE_NAME = "fleet_errors.log"
    private const val MAX_SIZE_BYTES = 512 * 1024L // 512 KB

    @Volatile private var logFile: File? = null

    fun initialize(context: Context) {
        logFile = File(context.filesDir, LOG_FILE_NAME)
    }

    fun e(tag: String, msg: String, throwable: Throwable? = null) {
        Log.e(tag, msg, throwable)
        appendToFile(tag, msg, throwable)
    }

    fun readErrors(): String = logFile?.takeIf { it.exists() }?.readText() ?: ""

    @Synchronized
    private fun appendToFile(tag: String, msg: String, throwable: Throwable?) {
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
            throwable?.let { t ->
                sb.append("    $t\n")
                t.stackTrace.forEach { element ->
                    sb.append("      at $element\n")
                }
            }
            file.appendText(sb.toString())
        } catch (e: Exception) {
            Log.e(TAG, "Failed to write to log file: ${e.message}")
        }
    }
}
