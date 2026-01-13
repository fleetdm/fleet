package com.fleetdm.agent

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.RestrictionsManager
import android.os.Bundle
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

class ManagedConfigurationRepository(private val context: Context) {
    private val restrictionsManager = context.getSystemService(RestrictionsManager::class.java)

    private val _configFlow = MutableStateFlow(getCurrentConfig())
    val configFlow: StateFlow<ManagedConfig> = _configFlow.asStateFlow()

    private val restrictionsReceiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context?, intent: Intent?) {
            if (intent?.action == Intent.ACTION_APPLICATION_RESTRICTIONS_CHANGED) {
                _configFlow.value = getCurrentConfig()
            }
        }
    }

    init {
        // Register receiver to listen for restriction changes
        val filter = IntentFilter(Intent.ACTION_APPLICATION_RESTRICTIONS_CHANGED)
        context.registerReceiver(restrictionsReceiver, filter)
    }

    private fun getCurrentConfig(): ManagedConfig {
        val restrictions = restrictionsManager.applicationRestrictions
        val certs = restrictions.getParcelableArray("certificate_templates", Bundle::class.java)?.map { bundle ->
            HostCertificate(
                id = bundle.getInt("id"),
                status = bundle.getString("status", ""),
                operation = bundle.getString("operation", HostCertificate.OPERATION_INSTALL),
                uuid = bundle.getString("uuid", ""),
            )
        }
        return ManagedConfig(
            serverUrl = restrictions.getString("server_url"),
            hostUUID = restrictions.getString("host_uuid"),
            hostCertificates = certs,
        )
    }

    fun cleanup() {
        context.unregisterReceiver(restrictionsReceiver)
    }
}

data class ManagedConfig(val serverUrl: String?, val hostUUID: String?, val hostCertificates: List<HostCertificate>?)
