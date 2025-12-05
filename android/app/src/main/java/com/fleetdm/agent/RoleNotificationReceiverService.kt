package com.fleetdm.agent

import android.util.Log
import com.google.android.managementapi.approles.AppRolesListener
import com.google.android.managementapi.approles.model.AppRolesSetRequest
import com.google.android.managementapi.approles.model.AppRolesSetResponse
import com.google.android.managementapi.notification.NotificationReceiverService

/**
 * Service to receive notifications from Android Device Policy (ADP) for COMPANION_APP role.
 * We need the service to force the app to run right after it is installed via MDM.
 */
class RoleNotificationReceiverService : NotificationReceiverService() {
    companion object {
        private const val TAG = "fleet-notification"
    }

    override fun getAppRolesListener(): AppRolesListener = object : AppRolesListener {
        override fun onAppRolesSet(request: AppRolesSetRequest): AppRolesSetResponse {
            Log.i(TAG, "App roles set by Android Device Policy")
            return AppRolesSetResponse.getDefaultInstance()
        }
    }
}
