@file:OptIn(ExperimentalMaterial3Api::class)

package com.fleetdm.agent

import android.app.admin.DevicePolicyManager
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Context.DEVICE_POLICY_SERVICE
import android.content.Context.RESTRICTIONS_SERVICE
import android.content.Intent
import android.content.RestrictionsManager
import android.content.pm.PackageInfo
import android.content.pm.PackageManager
import android.graphics.Color
import android.os.Bundle
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
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.core.net.toUri
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import com.fleetdm.agent.ui.theme.FleetTextDark
import com.fleetdm.agent.ui.theme.MyApplicationTheme
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

    var versionClicks by remember { mutableStateOf(0) }
    val installedCerts by CertificateOrchestrator.installedCertsFlow(context).collectAsStateWithLifecycle(initialValue = emptyMap())

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
                        val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
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

    val restrictionsManager = context.getSystemService(RESTRICTIONS_SERVICE) as RestrictionsManager
    val appRestrictions = restrictionsManager.applicationRestrictions
    val dpm = context.getSystemService(DEVICE_POLICY_SERVICE) as DevicePolicyManager

    val delegatedScopes = remember { dpm.getDelegatedScopes(null, context.packageName).toList() }
    val enrollmentSpecificID = remember { appRestrictions.getString("host_uuid")?.let { "****" + it.takeLast(4) } }
    val certTemplates = remember { CertificateOrchestrator.getCertificateTemplates(context) }
    val permissionsList = remember {
        val grantedPermissions = mutableListOf<String>()
        val packageInfo: PackageInfo = context.packageManager.getPackageInfo(context.packageName, PackageManager.GET_PERMISSIONS)
        packageInfo.requestedPermissions?.let {
            for (i in it.indices) {
                if ((
                        packageInfo.requestedPermissionsFlags?.get(i)
                            ?.and(PackageInfo.REQUESTED_PERMISSION_GRANTED)
                        ) != 0
                ) {
                    grantedPermissions.add(it[i])
                }
            }
        }
        grantedPermissions.toList()
    }
    val fleetBaseUrl = remember { appRestrictions.getString("server_url") }
    val baseUrl by ApiClient.baseUrlFlow.collectAsStateWithLifecycle(initialValue = null)
    val installedCerts by CertificateOrchestrator.installedCertsFlow(context).collectAsStateWithLifecycle(initialValue = emptyMap())

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
                KeyValue("certificate_templates", certTemplates?.map { "${it.id}:${it.operation}" }.toString())
                DebugCertificateList(certificates = installedCerts)
                PermissionList(
                    permissionsList = permissionsList,
                )
            }
        },
    )
}

@Composable
fun DebugCertificateList(certificates: CertStatusMap) {
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
fun CertificateList(modifier: Modifier = Modifier, certificates: CertStatusMap) {
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
            if (value.status == CertificateInstallStatus.INSTALLED) {
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
                    1 to CertificateInstallInfo(alias = "WIFI-1", status = CertificateInstallStatus.INSTALLED),
                    2 to CertificateInstallInfo(alias = "VPN-3", status = CertificateInstallStatus.FAILED),
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
                1 to CertificateInstallInfo(alias = "WIFI-1", status = CertificateInstallStatus.INSTALLED),
                2 to CertificateInstallInfo(alias = "VPN-3", status = CertificateInstallStatus.FAILED),
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
