package com.fleetdm.agent.osquery.tables

import android.app.admin.DevicePolicyManager
import android.content.Context
import android.content.RestrictionsManager
import android.os.Build
import android.os.UserManager
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext

class MdmStatusTable(private val context: Context) : TablePlugin {
    override val name: String = "mdm_status"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("has_device_owner"),
        ColumnDef("device_owner_package"),
        ColumnDef("has_work_profile"),
        ColumnDef("restrictions_present"),
        ColumnDef("enroll_secret_present"),
        ColumnDef("host_uuid_present"),
        ColumnDef("server_url_present"),
        ColumnDef("is_debug_build"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val dpm = context.getSystemService(Context.DEVICE_POLICY_SERVICE) as? DevicePolicyManager
        val um = context.getSystemService(Context.USER_SERVICE) as? UserManager
        val rm = context.getSystemService(Context.RESTRICTIONS_SERVICE) as? RestrictionsManager
        val restrictions = rm?.applicationRestrictions

        val hasDeviceOwner = dpm?.isDeviceOwnerApp(context.packageName) == true
        val ownerPkg = if (hasDeviceOwner) context.packageName else ""
        val hasWorkProfile =
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) um?.isManagedProfile == true else false

        val enrollPresent = !restrictions?.getString("enroll_secret").isNullOrBlank()
        val hostPresent = !restrictions?.getString("host_uuid").isNullOrBlank()
        val serverPresent = !restrictions?.getString("server_url").isNullOrBlank()
        val restrictionsPresent = restrictions != null && restrictions.keySet().isNotEmpty()

        return listOf(
            mapOf(
                "has_device_owner" to bool01(hasDeviceOwner),
                "device_owner_package" to ownerPkg,
                "has_work_profile" to bool01(hasWorkProfile),
                "restrictions_present" to bool01(restrictionsPresent),
                "enroll_secret_present" to bool01(enrollPresent),
                "host_uuid_present" to bool01(hostPresent),
                "server_url_present" to bool01(serverPresent),
                "is_debug_build" to bool01(com.fleetdm.agent.BuildConfig.DEBUG),
            ),
        )
    }

    private fun bool01(v: Boolean) = if (v) "1" else "0"
}
