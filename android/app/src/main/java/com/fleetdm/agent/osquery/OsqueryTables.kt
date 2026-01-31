package com.fleetdm.agent.osquery

import com.fleetdm.agent.osquery.core.TableRegistry
import com.fleetdm.agent.osquery.tables.AppPermissionsTable
import com.fleetdm.agent.osquery.tables.InstalledAppsTable
import com.fleetdm.agent.osquery.tables.OsVersionTable
import com.fleetdm.agent.osquery.tables.OsqueryInfoTable


object OsqueryTables {
    fun registerAll(context: android.content.Context) {
        TableRegistry.register(InstalledAppsTable(context))
        TableRegistry.register(AppPermissionsTable(context))
        TableRegistry.register(OsVersionTable())
        TableRegistry.register(OsqueryInfoTable(context))

    }
}


