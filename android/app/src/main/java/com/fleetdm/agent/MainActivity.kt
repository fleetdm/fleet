@file:OptIn(ExperimentalMaterial3Api::class)

package com.fleetdm.agent

import android.app.admin.DevicePolicyManager
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Intent
import android.content.RestrictionsManager
import android.content.pm.PackageInfo
import android.content.pm.PackageManager
import android.graphics.Color
import android.os.Bundle
import android.util.Log
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.Image
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.core.net.toUri
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.work.Constraints
import androidx.work.Data
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkInfo
import androidx.work.WorkManager
import com.fleetdm.agent.ui.theme.FleetTextDark
import com.fleetdm.agent.ui.theme.MyApplicationTheme
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.flow.map
import kotlinx.serialization.Serializable

const val CLICKS_TO_DEBUG = 8

@Serializable
object MainDestination

@Serializable
object DebugDestination

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge(
            statusBarStyle = SystemBarStyle.light(
                scrim = Color.TRANSPARENT,
                darkScrim = Color.TRANSPARENT,
            ),
        )

        setContent {
            MyApplicationTheme {
                AppNavigation()
            }
        }
    }
}

@Composable
fun AppNavigation() {
    val navController = rememberNavController()

    NavHost(
        navController = navController,
        startDestination = MainDestination,
    ) {
        composable<MainDestination> {
            MainScreen(
                onNavigateToDebug = { navController.navigate(DebugDestination) },
            )
        }

        composable<DebugDestination> {
            DebugScreen(
                onNavigateBack = { navController.navigateUp() },
            )
        }
    }
}

@Composable
fun MainScreen(onNavigateToDebug: () -> Unit) {
    val context = LocalContext.current
    val orchestrator = remember { AgentApplication.getCertificateOrchestrator(context) }

    var versionClicks by remember { mutableStateOf(0) }
    val installedCerts by orchestrator.installedCertsFlow(context).collectAsStateWithLifecycle(initialValue = emptyMap())

    Scaffold(
        modifier = Modifier.fillMaxSize(),
        content = { paddingValues ->
            Column(Modifier.padding(paddingValues = paddingValues)) {
                LogoHeader()
                HorizontalDivider()
                AboutFleet {
                    val intent = Intent(Intent.ACTION_VIEW, BuildConfig.INFO_URL.toUri())
                    if (intent.resolveActivity(context.packageManager) != null) {
                        // A browser is available, open the URL directly
                        context.startActivity(intent)
                    } else {
                        // A browser is not available in the work profile, display a toast with the URL
                        val toast = Toast.makeText(context, "Visit ${BuildConfig.INFO_URL}\nfor more information", Toast.LENGTH_LONG)
                        toast.show()
                    }
                }
                HorizontalDivider()
                CertificateList(certificates = installedCerts)
                AppVersion {
                    if (++versionClicks >= CLICKS_TO_DEBUG) {
                        onNavigateToDebug()
                    } else if (versionClicks == 1) {
                        val clipboard = context.getSystemService(ClipboardManager::class.java)
                            ?: error("ClipboardManager not available")
                        clipboard.setPrimaryClip(ClipData.newPlainText("", "Fleet Android Agent: ${BuildConfig.VERSION_NAME}"))
                        Toast.makeText(context, "Fleet Agent version copied", Toast.LENGTH_SHORT).show()
                    }
                }
            }
        },
    )
}

@Composable
fun DebugScreen(onNavigateBack: () -> Unit) {
    val context = LocalContext.current

    val dpm = context.getSystemService(DevicePolicyManager::class.java)
        ?: error("DevicePolicyManager not available")

    val orchestrator = remember { AgentApplication.getCertificateOrchestrator(context) }
    val delegatedScopes = remember { dpm.getDelegatedScopes(null, context.packageName).toList() }
    val managedConfigRepo = remember { ManagedConfigurationRepository(context) }
    val managedConfig by managedConfigRepo.configFlow.collectAsStateWithLifecycle(ManagedConfig(null, null, null))
    val enrollmentSpecificID = managedConfig.hostUUID?.let { "****" + it.takeLast(4) }
    val hostCertificates = managedConfig.hostCertificates
    val fleetBaseUrl = managedConfig.serverUrl
    val permissionsList = remember {
        val packageInfo = context.packageManager.getPackageInfo(context.packageName, PackageManager.GET_PERMISSIONS)
        val permissions = packageInfo.requestedPermissions ?: return@remember emptyList()
        val flags = packageInfo.requestedPermissionsFlags ?: return@remember emptyList()

        permissions.zip(flags.toList())
            .filter { (_, flag) -> flag and PackageInfo.REQUESTED_PERMISSION_GRANTED != 0 }
            .map { (permission, _) -> permission }
    }
    val baseUrl by ApiClient.baseUrlFlow.collectAsStateWithLifecycle(initialValue = null)
    val installedCerts by orchestrator.installedCertsFlow(context).collectAsStateWithLifecycle(initialValue = emptyMap())

    // Observe worker status
    val workManager = remember { WorkManager.getInstance(context) }
    val debugWorkInfoFlow = remember {
        workManager
            .getWorkInfosForUniqueWorkFlow("${CertificateEnrollmentWorker.WORK_NAME}_debug")
            .map { workInfos -> workInfos.firstOrNull() }
    }
    val debugWorkInfo by debugWorkInfoFlow.collectAsStateWithLifecycle(initialValue = null)

    // Observe periodic worker status
    val periodicWorkInfoFlow = remember {
        workManager
            .getWorkInfosForUniqueWorkFlow(CertificateEnrollmentWorker.WORK_NAME)
            .map { workInfos -> workInfos.firstOrNull() }
    }
    val periodicWorkInfo by periodicWorkInfoFlow.collectAsStateWithLifecycle(initialValue = null)

    Scaffold(
        modifier = Modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text("Debug Information") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = "Back",
                        )
                    }
                },
            )
        },
        content = { paddingValues ->
            Column(
                Modifier
                    .padding(paddingValues = paddingValues)
                    .verticalScroll(rememberScrollState()),
            ) {
                KeyValue("packageName", context.packageName)
                KeyValue("versionName", BuildConfig.VERSION_NAME)
                KeyValue("longVersionCode", BuildConfig.VERSION_CODE.toString())
                KeyValue("delegatedScopes", delegatedScopes.toString())
                KeyValue("host_uuid (MC)", enrollmentSpecificID)
                KeyValue("server_url (MC)", fleetBaseUrl)
                KeyValue("server_url (DS)", baseUrl)
                KeyValue("host_certificates", hostCertificates?.map { "${it.id}:${it.operation}" }.toString())
                Spacer(modifier = Modifier.height(16.dp))
                DebugCertificateList(certificates = installedCerts)
                PermissionList(permissionsList = permissionsList)
                ScheduledCertificateEnrollmentSection(workInfo = periodicWorkInfo)
                Spacer(modifier = Modifier.height(16.dp))
                CertificateEnrollmentDebugSection(
                    workManager = workManager,
                    workInfo = debugWorkInfo,
                )
                Spacer(modifier = Modifier.height(16.dp))
            }
        },
    )
}

@Composable
private fun ScheduledCertificateEnrollmentSection(workInfo: WorkInfo?) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
    ) {
        Text(
            text = "Scheduled Certificate Enrollment",
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.padding(bottom = 8.dp),
        )

        WorkerStatusDisplay(
            workInfo = workInfo,
            statusTextBuilder = { info ->
                if (info == null) "Worker not scheduled" else buildScheduledWorkerStatusText(info)
            },
        )
    }
}

@Composable
private fun CertificateEnrollmentDebugSection(workManager: WorkManager, workInfo: WorkInfo?) {
    val context = LocalContext.current

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
    ) {
        Text(
            text = "Debug Certificate Enrollment",
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.padding(bottom = 8.dp),
        )

        // Button to trigger enrollment
        Button(
            onClick = {
                val workRequest = OneTimeWorkRequestBuilder<CertificateEnrollmentWorker>()
                    .setInputData(
                        Data.Builder()
                            .putBoolean(CertificateEnrollmentWorker.KEY_IS_DEBUG, true)
                            .build(),
                    )
                    .setConstraints(
                        Constraints.Builder()
                            .setRequiredNetworkType(NetworkType.CONNECTED)
                            .build(),
                    )
                    .build()

                workManager.enqueueUniqueWork(
                    "${CertificateEnrollmentWorker.WORK_NAME}_debug",
                    ExistingWorkPolicy.REPLACE,
                    workRequest,
                )

                Log.d("DebugScreen", "Triggered one-time certificate enrollment (debug mode)")
            },
            modifier = Modifier.fillMaxWidth(),
            enabled = workInfo?.state != WorkInfo.State.RUNNING,
            colors = ButtonDefaults.buttonColors(
                containerColor = androidx.compose.ui.graphics.Color(0xFF1976D2), // Material Blue 700
                contentColor = androidx.compose.ui.graphics.Color.White,
                disabledContainerColor = androidx.compose.ui.graphics.Color(0xFFE0E0E0), // Light gray
                disabledContentColor = androidx.compose.ui.graphics.Color(0xFF9E9E9E), // Medium gray
            ),
        ) {
            Text(
                text = if (workInfo?.state == WorkInfo.State.RUNNING) {
                    "Enrollment Running..."
                } else {
                    "Trigger Certificate Enrollment"
                },
                color = androidx.compose.ui.graphics.Color.White,
            )
        }

        // Worker status display
        Spacer(modifier = Modifier.height(8.dp))
        WorkerStatusDisplay(
            workInfo = workInfo,
            statusTextBuilder = { info ->
                if (info == null) "No enrollment worker queued" else buildWorkerStatusText(info)
            },
        )
    }
}

@Composable
private fun WorkerStatusDisplay(workInfo: WorkInfo?, statusTextBuilder: (WorkInfo?) -> String) {
    val statusText = statusTextBuilder(workInfo)
    val backgroundColor = getWorkerStateColor(workInfo?.state)

    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        color = backgroundColor,
        shape = MaterialTheme.shapes.small,
    ) {
        Text(
            text = statusText,
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.padding(12.dp),
            fontFamily = FontFamily.Monospace,
        )
    }
}

@Composable
private fun getWorkerStateColor(state: WorkInfo.State?) = when (state) {
    WorkInfo.State.RUNNING -> MaterialTheme.colorScheme.primary.copy(alpha = 0.08f)
    WorkInfo.State.SUCCEEDED -> MaterialTheme.colorScheme.tertiary.copy(alpha = 0.08f)
    WorkInfo.State.FAILED -> MaterialTheme.colorScheme.error.copy(alpha = 0.08f)
    WorkInfo.State.ENQUEUED -> MaterialTheme.colorScheme.secondary.copy(alpha = 0.08f)
    WorkInfo.State.BLOCKED -> MaterialTheme.colorScheme.surfaceVariant
    WorkInfo.State.CANCELLED -> MaterialTheme.colorScheme.surfaceVariant
    null -> MaterialTheme.colorScheme.surface
}

private fun buildWorkerStatusText(workInfo: WorkInfo): String {
    val state = workInfo.state.name
    val runAttempt = workInfo.runAttemptCount
    val tags = workInfo.tags.joinToString(", ")

    return buildString {
        appendLine("Status: $state")
        appendLine("Run Attempt: $runAttempt")

        // Show next schedule time for periodic work
        if (workInfo.nextScheduleTimeMillis != Long.MAX_VALUE) {
            val nextRun = SimpleDateFormat("HH:mm:ss", Locale.US)
                .format(Date(workInfo.nextScheduleTimeMillis))
            appendLine("Next Run: $nextRun")
        }

        // Show output data if available
        if (workInfo.outputData.keyValueMap.isNotEmpty()) {
            appendLine("Output:")
            workInfo.outputData.keyValueMap.forEach { (key, value) ->
                appendLine("  $key: $value")
            }
        }

        // Show tags
        if (tags.isNotEmpty()) {
            appendLine("Tags: $tags")
        }
    }.trim()
}

private fun buildScheduledWorkerStatusText(workInfo: WorkInfo): String {
    val state = workInfo.state.name
    val now = System.currentTimeMillis()

    return buildString {
        appendLine("Status: $state")

        // Show next run time
        if (workInfo.nextScheduleTimeMillis != Long.MAX_VALUE) {
            val nextRunTime = workInfo.nextScheduleTimeMillis
            val timeUntilRun = nextRunTime - now

            val nextRunFormatted = SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.US)
                .format(Date(nextRunTime))
            appendLine("Next Run: $nextRunFormatted")

            // Show time until next run
            if (timeUntilRun > 0) {
                val minutes = TimeUnit.MILLISECONDS.toMinutes(timeUntilRun)
                val seconds = TimeUnit.MILLISECONDS.toSeconds(timeUntilRun) % 60
                appendLine("Time Until: ${minutes}m ${seconds}s")
            }
        }

        // Show run attempt count
        if (workInfo.runAttemptCount > 0) {
            appendLine("Run Attempt: ${workInfo.runAttemptCount}")
        }

        // Show output data if available
        if (workInfo.outputData.keyValueMap.isNotEmpty()) {
            appendLine("Last Run Output:")
            workInfo.outputData.keyValueMap.forEach { (key, value) ->
                appendLine("  $key: $value")
            }
        }
    }.trim()
}

@Composable
fun DebugCertificateList(certificates: CertificateStateMap) {
    Column {
        Text("certificate status:", fontWeight = FontWeight.Bold)
        certificates.forEach { (key, value) ->
            Row(modifier = Modifier.padding(bottom = 5.dp, start = 10.dp)) {
                Text(
                    text = key.toString(),
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.padding(end = 5.dp),
                )
                Column {
                    Text(text = "alias: ${value.alias}")
                    Text(text = "status: ${value.status}")
                    Text(text = "retries: ${value.retries}")
                    Text(text = "uuid: ${value.uuid}")
                    Text(text = "notBefore: ${value.notBefore}")
                    Text(text = "notAfter: ${value.notAfter}")
                    Text(text = "serial: ${value.serialNumber}")
                }
            }
        }
        HorizontalDivider()
    }
}

@Composable
fun PermissionList(modifier: Modifier = Modifier, permissionsList: List<String>) {
    Column(modifier = modifier) {
        Text(text = "permission list:", fontWeight = FontWeight.Bold)
        permissionsList.forEach {
            Row {
                Text(text = "- ", modifier = Modifier.padding(end = 8.dp))
                Text(text = it)
            }
        }
    }
}

@Composable
fun KeyValue(key: String, value: String?) {
    Text(
        buildAnnotatedString {
            withStyle(style = SpanStyle(fontWeight = FontWeight.Bold)) {
                append(key)
            }
            append(": $value")
        },
    )
    HorizontalDivider()
}

@Composable
fun AboutFleet(modifier: Modifier = Modifier, onLearnClick: () -> Unit = {}) {
    Column(modifier = modifier.padding(20.dp)) {
        Text(
            text = stringResource(R.string.app_description),
        )
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(4.dp),
            modifier = Modifier
                .padding(top = 10.dp)
                .clickable(onClick = onLearnClick),
        ) {
            Text(
                text = stringResource(R.string.learn_about_fleet),
                fontWeight = FontWeight.Bold,
                color = FleetTextDark,
            )
            Icon(imageVector = Icons.AutoMirrored.Default.ArrowForward, contentDescription = "forward arrow")
        }
    }
}

@Composable
fun LogoHeader(modifier: Modifier = Modifier) {
    Image(
        modifier = modifier.padding(20.dp),
        painter = painterResource(R.drawable.fleet_logo),
        contentDescription = stringResource(R.string.fleet_logo),
    )
}

@Composable
fun CertificateList(modifier: Modifier = Modifier, certificates: CertificateStateMap) {
    Column(modifier = modifier.padding(all = 20.dp)) {
        Text(
            text = stringResource(R.string.certificate_list_title),
            color = FleetTextDark,
            fontWeight = FontWeight.Bold,
        )
        certificates.ifEmpty {
            Text(text = stringResource(R.string.certificate_list_no_certificates))
        }
        certificates.forEach { (_, value) ->
            if (value.status == CertificateStatus.INSTALLED || value.status == CertificateStatus.INSTALLED_UNREPORTED) {
                Text(text = value.alias)
            }
        }
    }
}

@Composable
fun AppVersion(onClick: () -> Unit = {}) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
    ) {
        Column(
            modifier = Modifier
                .padding(horizontal = 20.dp),
        ) {
            Text(
                text = stringResource(R.string.app_version_title),
                color = FleetTextDark,
                fontWeight = FontWeight.Bold,
            )
            Text(text = BuildConfig.VERSION_NAME)
        }
    }
}

@Preview(showBackground = true)
@Composable
fun FleetScreenPreview() {
    MyApplicationTheme {
        Column {
            LogoHeader()
            HorizontalDivider()
            AboutFleet()
            HorizontalDivider()
            CertificateList(
                certificates = mapOf(
                    1 to CertificateState(alias = "WIFI-1", status = CertificateStatus.INSTALLED),
                    2 to CertificateState(alias = "VPN-3", status = CertificateStatus.FAILED),
                ),
            )
            AppVersion(onClick = {})
        }
    }
}

@Preview(showBackground = true)
@Composable
fun DebugCertificateListPreview() {
    MyApplicationTheme {
        DebugCertificateList(
            certificates = mapOf(
                1 to CertificateState(alias = "WIFI-1", status = CertificateStatus.INSTALLED),
                2 to CertificateState(alias = "VPN-3", status = CertificateStatus.FAILED),
            ),
        )
    }
}

@Preview(showBackground = true)
@Composable
fun AboutFleetPreview() {
    MyApplicationTheme {
        AboutFleet()
    }
}
