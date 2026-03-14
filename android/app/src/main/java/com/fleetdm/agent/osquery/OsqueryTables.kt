package com.fleetdm.agent.osquery

import com.fleetdm.agent.osquery.core.TableRegistry
import com.fleetdm.agent.osquery.tables.AppPermissionsTable
import com.fleetdm.agent.osquery.tables.InstalledAppsTable
import com.fleetdm.agent.osquery.tables.OsVersionTable
import com.fleetdm.agent.osquery.tables.OsqueryInfoTable
import com.fleetdm.agent.osquery.tables.CertificatesTable
import com.fleetdm.agent.osquery.tables.DeviceInfoTable
import com.fleetdm.agent.osquery.tables.NetworkInterfacesTable
import com.fleetdm.agent.osquery.tables.BatteryTable
import com.fleetdm.agent.osquery.tables.WifiNetworksTable
import com.fleetdm.agent.osquery.tables.SystemPropertiesTable
import com.fleetdm.agent.osquery.tables.AndroidLogcatTable
import com.fleetdm.agent.osquery.tables.TimeTable
import com.fleetdm.agent.osquery.tables.UptimeTable
import com.fleetdm.agent.osquery.tables.SystemInfoTable
import com.fleetdm.agent.osquery.tables.KernelInfoTable
import com.fleetdm.agent.osquery.tables.MemoryInfoTable
import com.fleetdm.agent.osquery.tables.ProcessesTable
import com.fleetdm.agent.osquery.tables.InterfaceAddressesTable
import com.fleetdm.agent.osquery.tables.RoutesTable
import com.fleetdm.agent.osquery.tables.UsersTable
import com.fleetdm.agent.osquery.tables.MountsTable
import com.fleetdm.agent.osquery.tables.CpuInfoTable



object OsqueryTables {
    fun registerAll(context: android.content.Context) {
        TableRegistry.register(InstalledAppsTable(context))
        TableRegistry.register(AppPermissionsTable(context))
        TableRegistry.register(OsVersionTable())
        TableRegistry.register(OsqueryInfoTable(context))
        TableRegistry.register(CertificatesTable())
        TableRegistry.register(DeviceInfoTable())
        TableRegistry.register(NetworkInterfacesTable(context))
        TableRegistry.register(BatteryTable(context))
        TableRegistry.register(WifiNetworksTable(context))
        TableRegistry.register(SystemPropertiesTable())
        TableRegistry.register(AndroidLogcatTable(context))
        TableRegistry.register(TimeTable())
        TableRegistry.register(UptimeTable())
        TableRegistry.register(SystemInfoTable(context))
        TableRegistry.register(KernelInfoTable())
        TableRegistry.register(MemoryInfoTable(context))
        TableRegistry.register(ProcessesTable(context))
        TableRegistry.register(InterfaceAddressesTable())
        TableRegistry.register(RoutesTable())
        TableRegistry.register(UsersTable())
        TableRegistry.register(MountsTable())
        TableRegistry.register(CpuInfoTable())

    }
}
