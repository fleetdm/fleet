package com.fleetdm.agent

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log

class BootReceiver : BroadcastReceiver() {
    companion object {
        private const val TAG = "fleet-boot"
    }

    override fun onReceive(context: Context?, intent: Intent?) {
        if (intent?.action == Intent.ACTION_BOOT_COMPLETED) {
            Log.i(TAG, "Device boot completed. Fleet Agent will initialize on app startup.")
            // Note: Certificate enrollment is now handled automatically in AgentApplication.onCreate()
            // when the app process starts. No additional action needed here.
        }
    }
}
