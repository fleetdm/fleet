package com.fleetdm.agent

import android.content.Context
import android.util.Log
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import com.fleetdm.agent.osquery.OsqueryQueryEngine
import java.util.concurrent.TimeUnit

class DistributedCheckinWorker(
    appContext: Context,
    params: WorkerParameters,
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        try {
            Log.d(TAG, "Distributed check-in: starting")

            val readResult = ApiClient.distributedRead()
            readResult.fold(
                onSuccess = { resp ->
                    val queries = resp.queries
                    Log.d(TAG, "Distributed check-in: received ${queries.size} query(ies)")

                    // 1) Log received queries
                    queries.forEach { (name, sql) ->
                        Log.i(TAG, "Distributed query [$name]:\n$sql")
                    }

                    // 2) Execute what we can, and write results back
                    val results = linkedMapOf<String, List<Map<String, String>>>()

                    for ((name, sql) in queries) {
                        if (sql.isBlank()) continue

                        try {
                            val rows = OsqueryQueryEngine.execute(sql)
                            results[name] = rows
                        } catch (e: Exception) {
                            val msg = e.message ?: e.javaClass.simpleName
                            Log.w(TAG, "Distributed query failed [$name]: $msg")

                            // "clear" unknown/unsupported queries so Fleet stops re-sending them
                            results[name] = emptyList()
                        }
                    }

                    if (results.isNotEmpty()) {
                        ApiClient.distributedWrite(results).fold(
                            onSuccess = {
                                Log.d(TAG, "Distributed check-in: wrote ${results.size} result set(s)")
                            },
                            onFailure = { err ->
                                Log.w(TAG, "Distributed check-in: write failed: ${err.message}")
                            },
                        )
                    }
                },
                onFailure = { err ->
                    Log.w(TAG, "Distributed check-in: read failed: ${err.message}")
                },
            )

            return Result.success()
        } finally {
            // Debug-only fast polling (15s). Release scheduling handled elsewhere (15m).
            if (BuildConfig.DEBUG) {
                scheduleNextDebug(applicationContext)
                Log.d(TAG, "Distributed check-in: scheduled next run in 15 seconds")
            }
        }
    }

    companion object {
        private const val TAG = "fleet-distributed"

        /** Single logical chain for debug polling */
        private const val WORK_NAME_DEBUG = "fleet_distributed_checkin_debug"

        fun scheduleNextDebug(context: Context) {
            val request = OneTimeWorkRequestBuilder<DistributedCheckinWorker>()
                .setInitialDelay(15, TimeUnit.SECONDS)
                .setConstraints(
                    Constraints.Builder()
                        .setRequiredNetworkType(NetworkType.CONNECTED)
                        .build()
                )
                .addTag(WORK_NAME_DEBUG)
                .build()

            WorkManager.getInstance(context)
                .beginUniqueWork(
                    WORK_NAME_DEBUG,
                    ExistingWorkPolicy.APPEND, // do NOT cancel running work
                    request
                )
                .enqueue()
        }
    }
}
