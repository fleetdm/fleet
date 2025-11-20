package com.fleetdm.agent

import android.app.Service
import android.app.admin.DevicePolicyManager
import android.content.Context
import android.content.Intent
import android.os.IBinder
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch
import org.json.JSONObject
import java.security.PrivateKey
import java.security.cert.Certificate
import java.security.cert.X509Certificate

/**
 * Service to handle SCEP enrollment and silent certificate installation using DevicePolicyManager.
 * Runs long-running tasks on the background IO thread via Coroutines.
 */
class CertificateService : Service() {
    private val TAG = "CertCompanionService"
    // Use a supervisor job for the service's lifecycle
    private val serviceJob = Job()
    private val serviceScope = CoroutineScope(Dispatchers.IO + serviceJob)

    // Data structure assumed to be pushed by the MDM for SCEP
    data class ScepConfig(
        val url: String,
        val challenge: String,
        val alias: String, // Added alias for silent installation
        val subject: String? = null
    )

    // Structure for SCEP result, holding the key and certificate(s)
    data class ScepResult(
        val privateKey: PrivateKey,
        val certificateChain: Array<Certificate>
    )


    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val certDataJson = intent?.getStringExtra("CERT_DATA")

        if (certDataJson != null) {
            // Launch the SCEP process in a coroutine on the IO dispatcher
            serviceScope.launch {
                try {
                    val scepConfig = parseScepConfig(certDataJson)
                    Log.d(TAG, "Parsed SCEP URL: ${scepConfig.url}")

                    // Step 1: Execute the SCEP enrollment process
                    val result = scepEnrollment(scepConfig)

                    if (result != null) {
                        Log.i(TAG, "SCEP enrollment succeeded. Performing silent installation.")

                        // Step 2: Perform the silent installation using DPM
                        installCertificateSilently(
                            scepConfig.alias,
                            result.privateKey,
                            result.certificateChain
                        )
                    } else {
                        Log.e(TAG, "SCEP enrollment failed or returned empty data.")
                    }
                } catch (e: Exception) {
                    Log.e(TAG, "Certificate installation failed due to error: ${e.message}", e)
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
     * Parses the JSON payload from the MDM into a structured configuration object.
     */
    private fun parseScepConfig(jsonString: String): ScepConfig {
        return try {
            val json = JSONObject(jsonString)
            ScepConfig(
                url = json.getString("scep_url"),
                challenge = json.getString("challenge"),
                alias = json.getString("alias"),
                subject = json.optString("subject", null)
            )
        } catch (e: Exception) {
            Log.e(TAG, "Failed to parse SCEP configuration JSON: ${e.message}")
            ScepConfig("", "", "default_alias", "")
        }
    }

    /**
     * !!! SIMPLIFIED PLACEHOLDER: YOUR SCEP CLIENT IMPLEMENTATION GOES HERE !!!
     * * In a real app, this function would handle:
     * 1. KeyPair Generation (using KeyGenParameterSpec and KeyPairGenerator)
     * 2. Certificate Signing Request (CSR) creation.
     * 3. Network communication with the SCEP server to get the PKCS#7 response.
     * 4. Parsing the response into a PrivateKey object and an Array of X509Certificate objects.
     * * @return The resulting ScepResult object containing the key and chain, or null if enrollment fails.
     */
    private fun scepEnrollment(config: ScepConfig): ScepResult? {
        Log.w(TAG, "--- SCEP Enrollment Placeholder Running ---")
        Log.w(TAG, "Your actual SCEP client library must replace this function.")

        // This MUST return a valid PrivateKey and Certificate array for installation to work.
        // Returning null to simulate failure for now.
        return null
    }

    /**
     * Performs a silent installation of the KeyPair using the delegated CERT_INSTALL scope.
     * This method requires NO user interaction on modern managed devices (API 18+).
     */
    private fun installCertificateSilently(
        alias: String,
        privateKey: PrivateKey,
        certificateChain: Array<Certificate>
    ) {
        val dpm = getSystemService(Context.DEVICE_POLICY_SERVICE) as DevicePolicyManager

        // The admin component is null because the caller is a DELEGATED application,
        // not the Device Policy Controller itself. The DPM recognizes the delegation
        // via the calling package's UID and the granted CERT_INSTALL scope.
        val success = dpm.installKeyPair(
            null,
            privateKey,
            certificateChain,
            alias
        )

        if (success) {
            Log.i(TAG, "Certificate successfully installed silently with alias: $alias")
        } else {
            Log.e(TAG, "Silent certificate installation failed. Check MDM policy and delegation status.")
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