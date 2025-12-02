package com.fleetdm.agent

import android.app.Service
import android.content.Intent
import android.os.IBinder
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch

/**
 * Service to handle certificate enrollment operations.
 *
 * This is a thin wrapper around CertificateOrchestrator that provides Android Service
 * lifecycle management. The actual certificate enrollment logic (API calls, SCEP enrollment,
 * and installation) is delegated to CertificateOrchestrator.
 */
class CertificateService : Service() {
    private val TAG = "CertificateService"

    // Use a supervisor job for the service's lifecycle
    private val serviceJob = Job()
    private val serviceScope = CoroutineScope(Dispatchers.IO + serviceJob)

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val certificateId = intent?.getIntExtra("CERTIFICATE_ID", -1) ?: -1

        if (certificateId > 0) {
            // Launch the certificate enrollment in a coroutine on the IO dispatcher
            serviceScope.launch {
                try {
                    val result = CertificateOrchestrator.enrollCertificate(
                        context = applicationContext,
                        certificateId = certificateId
                    )

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
            Log.w(TAG, "Service started without valid 'CERTIFICATE_ID' extra.")
            stopSelf(startId)
        }
        return START_NOT_STICKY
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
