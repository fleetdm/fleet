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
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.core.net.toUri
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import com.fleetdm.agent.ui.theme.FleetTextDark
import com.fleetdm.agent.ui.theme.MyApplicationTheme
import com.fleetdm.agent.device.DeviceIdManager
import com.fleetdm.agent.device.Storage
import kotlinx.serialization.Serializable

const val CLICKS_TO_DEBUG = 8

@Serializable object MainDestination
@Serializable object DebugDestination
@Serializable object LogsDestination

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
    }
}

@Composable
fun MainScreen(onNavigateToDebug: () -> Unit) {
    val context = LocalContext.current
    val orchestrator = remember { AgentApplication.getCertificateOrchestrator(context) }

    val deviceId = remember {
        Storage.init(context)
        DeviceIdManager.getOrCreateDeviceId()
    }

    var versionClicks by remember { mutableStateOf(0) }
    val installedCerts by orchestrator
        .installedCertsFlow(context)
        .collectAsStateWithLifecycle(initialValue = emptyMap())

    Scaffold(
        modifier = Modifier.fillMaxSize(),
        content = { paddingValues ->
            Column(
                Modifier
                    .padding(paddingValues)
                    .verticalScroll(rememberScrollState())
            ) {
                LogoHeader()
                HorizontalDivider()
                AboutFleet {
                    val intent = Intent(Intent.ACTION_VIEW, BuildConfig.INFO_URL.toUri())
                    context.startActivity(intent)
                }
                HorizontalDivider()
                CertificateList(certificates = installedCerts)
                AppVersion {
                    if (++versionClicks >= CLICKS_TO_DEBUG) {
                        onNavigateToDebug()
                    }
                }
                DeviceIdRow(deviceId = deviceId) {
                    val clipboard = context.getSystemService(ClipboardManager::class.java)
                    clipboard.setPrimaryClip(
                        ClipData.newPlainText("Device ID", deviceId)
                    )
                    Toast.makeText(context, "Device ID copied", Toast.LENGTH_SHORT).show()
                }
            }
        },
    )
}

@Composable
fun DeviceIdRow(deviceId: String, onClick: () -> Unit = {}) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
    ) {
        Column(
            modifier = Modifier.padding(horizontal = 20.dp, vertical = 14.dp),
        ) {
            Text(
                text = "Device ID",
                color = FleetTextDark,
                fontWeight = FontWeight.Bold,
            )
            Text(text = deviceId)
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
            modifier = Modifier.padding(horizontal = 20.dp),
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

@Composable
fun AboutFleet(onLearnClick: () -> Unit = {}) {
    Column(Modifier.padding(20.dp)) {
        Text(text = stringResource(R.string.app_description))
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .padding(top = 10.dp)
                .clickable(onClick = onLearnClick),
        ) {
            Text(
                text = stringResource(R.string.learn_about_fleet),
                fontWeight = FontWeight.Bold,
                color = FleetTextDark,
            )
            Icon(
                imageVector = Icons.AutoMirrored.Default.ArrowForward,
                contentDescription = null,
            )
        }
    }
}

@Composable
fun LogoHeader() {
    Image(
        modifier = Modifier.padding(20.dp),
        painter = painterResource(R.drawable.fleet_logo),
        contentDescription = null,
    )
}

@Composable
fun CertificateList(certificates: CertificateStateMap) {
    Column(Modifier.padding(20.dp)) {
        Text(
            text = stringResource(R.string.certificate_list_title),
            color = FleetTextDark,
            fontWeight = FontWeight.Bold,
        )
        certificates.ifEmpty {
            Text(text = stringResource(R.string.certificate_list_no_certificates))
        }
        certificates.forEach { (_, value) ->
            if (value.status == CertificateStatus.INSTALLED ||
                value.status == CertificateStatus.INSTALLED_UNREPORTED
            ) {
                Text(text = value.alias)
            }
        }
    }
}
