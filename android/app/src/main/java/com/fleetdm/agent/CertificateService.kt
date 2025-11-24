package com.fleetdm.agent

import android.app.Service
import android.app.admin.DevicePolicyManager
import android.content.Context
import android.content.Intent
import android.os.IBinder
import android.util.Log
import com.fleetdm.agent.scep.ScepClientImpl
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch
import java.security.PrivateKey
import java.security.cert.Certificate

/**
 * Service to handle SCEP enrollment and silent certificate installation using DevicePolicyManager.
 * Runs long-running tasks on the background IO thread via Coroutines.
 *
 * This is a thin wrapper around CertificateEnrollmentHandler that provides Android-specific
 * lifecycle management and certificate installation.
 */
class CertificateService : Service() {
    private val TAG = "CertCompanionService"

    // Use a supervisor job for the service's lifecycle
    private val serviceJob = Job()
    private val serviceScope = CoroutineScope(Dispatchers.IO + serviceJob)

    // Enrollment handler with Android-specific certificate installer
    private val enrollmentHandler = CertificateEnrollmentHandler(
        scepClient = ScepClientImpl(),
        certificateInstaller = AndroidCertificateInstaller()
    )

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val certDataJson = intent?.getStringExtra("CERT_DATA")

        if (certDataJson != null) {
            // Launch the SCEP process in a coroutine on the IO dispatcher
            serviceScope.launch {
                try {
                    val result = enrollmentHandler.handleEnrollment(certDataJson)

                    when (result) {
                        is CertificateEnrollmentHandler.EnrollmentResult.Success -> {
                            Log.i(TAG, "Certificate successfully enrolled and installed with alias: ${result.alias}")
                        }
                        is CertificateEnrollmentHandler.EnrollmentResult.Failure -> {
                            Log.e(TAG, "Certificate enrollment failed: ${result.reason}", result.exception)
                        }
                    }
                } catch (e: Exception) {
                    Log.e(TAG, "Unexpected error during certificate enrollment: ${e.message}", e)
                } finally {
                    // Stop the service when work is done, regardless of success/failure
                    stopSelf(startId)
                }
            }
        } else {
            Log.w(TAG, "Service started without 'CERT_DATA' extra.")
            stopSelf(startId)
        }
        return START_NOT_STICKY
    }

    /**
     * Android-specific certificate installer using DevicePolicyManager.
     */
    inner class AndroidCertificateInstaller : CertificateEnrollmentHandler.CertificateInstaller {
        override fun installCertificate(
            alias: String,
            privateKey: PrivateKey,
            certificateChain: Array<Certificate>
        ): Boolean {
            val dpm = getSystemService(Context.DEVICE_POLICY_SERVICE) as DevicePolicyManager

            // The admin component is null because the caller is a DELEGATED application,
            // not the Device Policy Controller itself. The DPM recognizes the delegation
            // via the calling package's UID and the granted CERT_INSTALL scope.
            val success = dpm.installKeyPair(
                null,
                privateKey,
                certificateChain,
                alias,
                true, // requestAccess: allows user confirmation if needed
            )

            if (success) {
                Log.i(TAG, "Certificate successfully installed with alias: $alias")
            } else {
                Log.e(TAG, "Certificate installation failed. Check MDM policy and delegation status.")
            }

            return success
        }
    }

    override fun onBind(intent: Intent?): IBinder? {
        return null // Not a bound service
    }

    override fun onDestroy() {
        super.onDestroy()
        // Cancel the coroutine scope when the service is destroyed to prevent leaks
        serviceJob.cancel()
    }
}
