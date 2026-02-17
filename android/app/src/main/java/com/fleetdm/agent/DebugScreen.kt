package com.fleetdm.agent

import android.app.admin.DevicePolicyManager
import android.content.ClipData
import android.content.ClipboardManager
import android.content.RestrictionsManager
import android.content.pm.PackageInfo
import android.content.pm.PackageManager
import android.widget.Toast
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.fleetdm.agent.ui.theme.MyApplicationTheme
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json


private val jsonPretty = Json { prettyPrint = true }

@Composable
fun DebugScreen(onNavigateBack: () -> Unit) {
    val context = LocalContext.current

    val restrictionsManager = context.getSystemService(RestrictionsManager::class.java)
        ?: error("RestrictionsManager not available")
    val appRestrictions = restrictionsManager.applicationRestrictions
    val dpm = context.getSystemService(DevicePolicyManager::class.java)
        ?: error("DevicePolicyManager not available")

    val orchestrator = remember { AgentApplication.getCertificateOrchestrator(context) }
    val delegatedScopes = remember { dpm.getDelegatedScopes(null, context.packageName).toList() }
    val enrollmentSpecificID = remember { appRestrictions.getString("host_uuid")?.let { "****" + it.takeLast(4) } }
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

    val margin = Modifier.padding(horizontal = 20.dp)

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
                actions = {
                    IconButton(onClick = {
                        val clipboard = context.getSystemService(ClipboardManager::class.java)
                            ?: error("ClipboardManager not available")
                        val info = DebugInformation(
                            packageName = context.packageName,
                            versionName = BuildConfig.VERSION_NAME,
                            longVersionCode = BuildConfig.VERSION_CODE,
                            delegatedScopes = delegatedScopes,
                            serverUrl = baseUrl,
                            certificateStatus = installedCerts,
                            permissionList = permissionsList,
                        )
                        val infoJson = jsonPretty.encodeToString(info)
                        clipboard.setPrimaryClip(ClipData.newPlainText("fleet debug information", infoJson))
                        Toast.makeText(context, "Debug info copied", Toast.LENGTH_SHORT).show()
                    }) {
                        Icon(Icons.Filled.ContentCopy, contentDescription = "copy debug information")
                    }
                },
            )
        },
        content = { paddingValues ->
            Column(
                Modifier
                    .padding(paddingValues = paddingValues)
                    .verticalScroll(rememberScrollState())
            ) {
                KeyValue(modifier = margin, key = "Package name", value = context.packageName)
                KeyValue(modifier = margin, key = "Version name", value = BuildConfig.VERSION_NAME)
                KeyValue(modifier = margin, key = "Long version code", value = BuildConfig.VERSION_CODE.toString())
                KeyValue(modifier = margin, key = "Delegated scopes", value = delegatedScopes.toString())
                KeyValue(modifier = margin, key = "Host UUID", value = enrollmentSpecificID)
                KeyValue(modifier = margin, key = "Server URL", value = baseUrl)
                DebugCertificateList(modifier = margin, certificates = installedCerts)
                PermissionList(
                    modifier = margin,
                    permissionsList = permissionsList,
                )
            }
        },
    )
}

@Composable
fun DebugCertificateList(modifier: Modifier = Modifier, certificates: CertificateStateMap) {
    Column {
        Column(modifier = modifier) {
            Text("Certificate states:", fontWeight = FontWeight.Bold)
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
        }
        HorizontalDivider()
    }
}

@Composable
fun PermissionList(modifier: Modifier = Modifier, permissionsList: List<String>) {
    Column(modifier = modifier) {
        Text(text = "Permission list:", fontWeight = FontWeight.Bold)
        permissionsList.forEach {
            Row {
                Text(text = "- ", modifier = Modifier.padding(end = 8.dp))
                Text(text = it)
            }
        }
    }
}

@Composable
fun KeyValue(modifier: Modifier = Modifier, key: String, value: String?) {
    Text(
        modifier = modifier,
        text = buildAnnotatedString {
            withStyle(style = SpanStyle(fontWeight = FontWeight.Bold)) {
                append(key)
            }
            append(": $value")
        },
    )
    HorizontalDivider()
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

@Serializable
data class DebugInformation (
    @SerialName("package_name")
    val packageName: String,
    @SerialName("version_name")
    val versionName: String,
    @SerialName("long_version_code")
    val longVersionCode: Int,
    @SerialName("delegated_scopes")
    val delegatedScopes: List<String>,
    @SerialName("server_url")
    val serverUrl: String?,
    @SerialName("certificate_status")
    val certificateStatus: CertificateStateMap,
    @SerialName("permission_list")
    val permissionList: List<String>
)
