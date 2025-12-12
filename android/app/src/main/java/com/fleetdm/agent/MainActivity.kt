package com.fleetdm.agent

import android.app.admin.DevicePolicyManager
import android.content.Context
import android.content.pm.PackageInfo
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.provider.Settings
import android.util.Log
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.datastore.preferences.core.stringPreferencesKey
import com.fleetdm.agent.ui.theme.MyApplicationTheme
import java.security.KeyStore
import java.security.cert.X509Certificate
import java.util.Date
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        // 1. Fetch the Managed Configuration (Application Restrictions)
        val restrictionsManager = getSystemService(Context.RESTRICTIONS_SERVICE) as android.content.RestrictionsManager
        val appRestrictions = restrictionsManager.applicationRestrictions
        val dpm = getSystemService(Context.DEVICE_POLICY_SERVICE) as DevicePolicyManager

        setContent {
            val enrollSecret by remember { mutableStateOf(appRestrictions.getString("enroll_secret")) }
            val delegatedScopes by remember { mutableStateOf(dpm.getDelegatedScopes(null, packageName).toList()) }
            val delegatedCertScope by remember {
                mutableStateOf(delegatedScopes.contains(DevicePolicyManager.DELEGATION_CERT_INSTALL))
            }
            val androidID by remember { mutableStateOf(Settings.Secure.getString(contentResolver, Settings.Secure.ANDROID_ID)) }
            val enrollmentSpecificID by remember { mutableStateOf(appRestrictions.getString("host_uuid")) }
            val certIds by remember { mutableStateOf(CertificateOrchestrator.getCertificateIDs(this)) }
            val permissionsList by remember {
                val grantedPermissions = mutableListOf<String>()
                val packageInfo: PackageInfo = packageManager.getPackageInfo(packageName, PackageManager.GET_PERMISSIONS)
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
                mutableStateOf(grantedPermissions.toList())
            }
            val fleetBaseUrl by remember {
                mutableStateOf(appRestrictions.getString("server_url"))
            }
            var installedCertificates: List<CertificateInfo> by remember { mutableStateOf(listOf()) }
            val apiKey by ApiClient.apiKeyFlow.collectAsState(initial = null)
            val baseUrl by ApiClient.baseUrlFlow.collectAsState(initial = null)
            val allegedInstalledCerts by CertificateOrchestrator.installedCertsFlow(this).collectAsState(initial = "")

            LaunchedEffect(Unit) {
                installedCertificates = listKeystoreCertificates()
            }

            MyApplicationTheme {
                Scaffold(
                    modifier = Modifier.fillMaxSize(),
                    content = { padding ->
                        Column(
                            modifier = Modifier.padding(padding).verticalScroll(rememberScrollState()),
                        ) {
                            StatusScreen()
                            KeyValue("packageName", packageName)
                            KeyValue("versionName", packageManager.getPackageInfo(packageName, 0).versionName)
                            KeyValue("longVersionCode", packageManager.getPackageInfo(packageName, 0).longVersionCode.toString())
                            KeyValue("enroll_secret", enrollSecret)
                            KeyValue("delegatedScopes", delegatedScopes.toString())
                            KeyValue("delegated cert scope", delegatedCertScope.toString())
                            KeyValue("android id", androidID)
                            KeyValue("host_uuid (MC)", enrollmentSpecificID)
                            KeyValue("server_url (MC)", fleetBaseUrl)
                            KeyValue("orbit_node_key (datastore)", apiKey)
                            KeyValue("base_url (datastore)", baseUrl)
                            KeyValue("certificate_templates->id", certIds.toString())
                            KeyValue("alleged_installed", allegedInstalledCerts.toString())
                            PermissionList(
                                permissionsList = permissionsList,
                            )
                            CertificateList(certificateList = installedCertificates)
                        }
                    },
                )
            }
        }
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
fun CertificateList(modifier: Modifier = Modifier, certificateList: List<CertificateInfo>) {
    Column(modifier = modifier) {
        Text("certificate list:")
        HorizontalDivider()
        certificateList.forEach {
            Text(it.alias)
            HorizontalDivider()
        }
    }
}

@Composable
fun StatusScreen(modifier: Modifier = Modifier) {
    val tag = "CertStatusScreen"
    Log.i(tag, "this is a log!")

    Greeting(
        name = "banana frog",
        modifier = modifier,
    )
}

@Composable
fun Greeting(name: String, modifier: Modifier = Modifier) {
    Text(
        text = "Hello $name!",
        modifier = modifier,
    )
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

suspend fun listKeystoreCertificates(): List<CertificateInfo> = withContext(Dispatchers.IO) {
    try {
        val keyStore = KeyStore.getInstance("AndroidKeyStore")
        keyStore.load(null)

        val aliases = keyStore.aliases().toList()

        aliases.mapNotNull { alias ->
            val cert = keyStore.getCertificate(alias) as? X509Certificate
            cert?.let {
                CertificateInfo(
                    alias = alias,
                    subject = it.subjectDN.name,
                    issuer = it.issuerDN.name,
                    notBefore = it.notBefore,
                    notAfter = it.notAfter,
                )
            }
        }
    } catch (e: Exception) {
        Log.e("Certificate", "Error listing keystore certificates", e)
        emptyList()
    }
}

data class CertificateInfo(val alias: String, val subject: String, val issuer: String, val notBefore: Date, val notAfter: Date)

@Preview(showBackground = true)
@Composable
fun GreetingPreview() {
    MyApplicationTheme {
        Greeting("Android")
    }
}
