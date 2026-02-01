package com.fleetdm.agent.osquery.tables

import android.os.Build
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext
import java.util.Locale

class OsVersionTable : TablePlugin {
    override val name: String = "os_version"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("name"),
        ColumnDef("version"),
        ColumnDef("major"),
        ColumnDef("minor"),
        ColumnDef("patch"),
        ColumnDef("build"),
        ColumnDef("platform"),
        ColumnDef("arch"),
        ColumnDef("security_patch"),
    )


    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val versionName = Build.VERSION.RELEASE ?: ""
        val major = Build.VERSION.SDK_INT.toString()

        val buildId = Build.ID ?: ""
        val platform = "android"
        val osName = "Android"

        val arch = Build.SUPPORTED_ABIS.firstOrNull() ?: ""
        val securityPatch = Build.VERSION.SECURITY_PATCH ?: ""

        val (minor, patch) = parseMinorPatch(versionName)

        return listOf(
            mapOf(
                "name" to osName,
                "version" to versionName,
                "major" to major,
                "minor" to minor,
                "patch" to patch,
                "build" to buildId,
                "platform" to platform,
                "arch" to arch,
                "security_patch" to securityPatch,
            ),
        )
    }


    private fun parseMinorPatch(version: String): Pair<String, String> {
        // Android version often looks like "14" or "13" or "12.1"
        // We keep major as SDK_INT, and try to extract minor and patch from RELEASE if present.
        val parts = version.trim().split(".")
            .map { it.trim() }
            .filter { it.isNotEmpty() }

        val minor = parts.getOrNull(1)?.takeWhile { it.isDigit() } ?: "0"
        val patch = parts.getOrNull(2)?.takeWhile { it.isDigit() } ?: "0"
        return minor.lowercase(Locale.ROOT) to patch.lowercase(Locale.ROOT)
    }
}
