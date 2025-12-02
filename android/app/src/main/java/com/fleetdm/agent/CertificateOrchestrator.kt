package com.fleetdm.agent

import android.app.admin.DevicePolicyManager
import android.content.Context
import android.os.Bundle
import android.util.Log
import androidx.core.content.ContextCompat.getSystemService
import com.fleetdm.agent.scep.ScepClientImpl
import java.security.PrivateKey
import java.security.cert.Certificate

object CertificateOrchestrator {
    fun getCertificateIDs(context: Context): List<Int>? {
        val restrictionsManager = context.getSystemService(Context.RESTRICTIONS_SERVICE) as android.content.RestrictionsManager
        val appRestrictions = restrictionsManager.applicationRestrictions

        val certRequestList = appRestrictions.getParcelableArray("certificates", Bundle::class.java)?.toList()
        return certRequestList?.map { bundle -> bundle.getInt("certificate_id") }
    }


}

