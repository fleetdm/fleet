package com.fleetdm.agent.osquery

import com.fleetdm.agent.osquery.core.TableRegistry
import com.fleetdm.agent.osquery.tables.AppPermissionsTable
import com.fleetdm.agent.osquery.tables.InstalledAppsTable
import com.fleetdm.agent.osquery.tables.OsVersionTable
import com.fleetdm.agent.osquery.tables.OsqueryInfoTable
import com.fleetdm.agent.osquery.tables.CertificatesTable
import com.fleetdm.agent.osquery.tables.DeviceInfoTable
import com.fleetdm.agent.osquery.tables.NetworkInterfacesTable


object OsqueryTables {
    fun registerAll(context: android.content.Context) {
        TableRegistry.register(InstalledAppsTable(context))
        TableRegistry.register(AppPermissionsTable(context))
        TableRegistry.register(OsVersionTable())
        TableRegistry.register(OsqueryInfoTable(context))
        TableRegistry.register(CertificatesTable())
        TableRegistry.register(DeviceInfoTable())
        TableRegistry.register(NetworkInterfacesTable(context))


    }
}


