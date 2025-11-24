package com.fleetdm.agent

import android.app.admin.DevicePolicyManager
import android.bluetooth.BluetoothClass
import android.content.Context
import android.content.RestrictionsManager
import android.content.pm.PackageInfo
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.provider.Settings
import android.security.KeyChain
import android.util.Log
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.LargeTopAppBar
import androidx.compose.material3.MediumTopAppBar
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.tooling.preview.Preview
import com.fleetdm.agent.ui.theme.MyApplicationTheme
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.toMutableStateList
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import java.net.HttpURLConnection
import java.net.URL
import java.security.KeyStore
import java.security.cert.X509Certificate
import java.util.Date

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        // 1. Fetch the Managed Configuration (Application Restrictions)
        val restrictionsManager = getSystemService(Context.RESTRICTIONS_SERVICE) as android.content.RestrictionsManager
        val appRestrictions = restrictionsManager.applicationRestrictions
        val dpm = getSystemService(Context.DEVICE_POLICY_SERVICE) as DevicePolicyManager

        ApiClient.initialize(this)

        setContent {
            val enrollSecret by remember { mutableStateOf(appRestrictions.getString("enrollSecret")) }
            val delegatedScopes by remember { mutableStateOf(dpm.getDelegatedScopes(null, packageName)) }
            val delegatedCertScope by remember {
                mutableStateOf(delegatedScopes.contains(
                    DevicePolicyManager.DELEGATION_CERT_INSTALL)
                )
            }
            val androidID by remember { mutableStateOf(Settings.Secure.getString(contentResolver, Settings.Secure.ANDROID_ID)) }
            val enrollmentSpecificID by remember { mutableStateOf(appRestrictions.getString("enrollmentSpecificID")) }
            val permissionsList by remember {
                val grantedPermissions = mutableListOf<String>()
                val packageInfo: PackageInfo = packageManager.getPackageInfo(packageName, PackageManager.GET_PERMISSIONS)
                packageInfo.requestedPermissions?.let {
                    for (i in it.indices) {
                        if ((packageInfo.requestedPermissionsFlags?.get(i)
                                ?.and(PackageInfo.REQUESTED_PERMISSION_GRANTED)) != 0) {
                            grantedPermissions.add(it[i])
                        }
                    }
                }
                mutableStateOf(grantedPermissions)
            }
            val fleetBaseUrl by remember {
                mutableStateOf(appRestrictions.getString("fleetBaseUrl"))
            }
//            var enrollUrl by remember {
//                val buildEnroll = fleetBaseUrl?.let { url ->
//                    Uri.parse(url)
//                        .buildUpon()
//                        .appendPath("api")
//                        .appendPath("fleet")
//                        .appendPath("orbit")
//                        .appendPath("enroll")
//                        .build()
//                        .toString()
//                }
//                mutableStateOf(buildEnroll)
//            }
            var clicks by remember { mutableStateOf(0) }
            var respBody by remember { mutableStateOf("not sent yet")}
            var enrollBody by remember {
//                val body = if (enrollUrl == null) {
//                    "no enroll url"
//                } else {
//                    "not enroll"
//                }
                mutableStateOf("not enrolled")
            }
            var installedCertificates: List<CertificateInfo> by remember { mutableStateOf(listOf()) }
            val scope = rememberCoroutineScope()

            LaunchedEffect(Unit) {
                installedCertificates = listKeystoreCertificates()
            }

            MyApplicationTheme {
                Scaffold(
                    modifier = Modifier.fillMaxSize(),
                    content = { padding ->
                        Column(
                            modifier = Modifier.padding(padding).verticalScroll(rememberScrollState())
                        ) {
                            StatusScreen(
                                dpm = dpm,
                            )
                            Text(text = "packageName: $packageName")
                            Text(text = "enrollSecret: $enrollSecret")
                            Text(text = "delegatedScopes: $delegatedScopes")
                            Text(text = "delegated cert scope: $delegatedCertScope")
                            Text(text = "android id: $androidID")
                            Text(text = "enrollmentSpecificID (from RM): $enrollmentSpecificID")
                            Text(text = "fleetBaseUrl: $fleetBaseUrl")
                            PermissionList(
                                modifier = Modifier.padding(10.dp),
                                permissionsList = permissionsList
                            )
                            Button(onClick = { clicks++ }) { Text("Clicks: $clicks") }
                            Button(onClick = {
                                scope.launch {
                                    respBody = "launched!"
                                    try {
                                        val resp = makeGetRequest("https://example.com")
                                        respBody = resp
                                    } catch (e: Exception) {
                                        respBody = e.toString()
                                    }
                                }
                            }) { Text("make request") }
                            Text(text = respBody)
                            Button(onClick = {
                                scope.launch {
                                    enrollBody = "launched!!"
                                    if (enrollSecret == null) {
                                        enrollBody = "no enroll secret"
                                    }
                                    if (fleetBaseUrl == null) {
                                        enrollBody = "no fleet URL"
                                    }
                                    try {
                                        Log.d("main_activity", "sending request!")
                                        val resp = ApiClient.enroll(
                                            baseUrl = fleetBaseUrl ?: "",
                                            enrollSecret = enrollSecret ?: "",
                                            hardwareUUID = enrollmentSpecificID ?: "",
                                            computerName = Build.MODEL,
                                        )
//                                        val resp = makePostRequest(
//                                            enrollUrl.toString(),
//                                            "",
//                                        )
                                        enrollBody = resp.toString()
                                    } catch (e: Exception) {
                                        enrollBody = e.toString()
                                    }
                                }
                            }) { Text("enroll") }
                            Text(enrollBody)
                            CertificateList(certificateList = installedCertificates)
                        }
                    }
                )
            }
        }
    }
}

@Composable
fun PermissionList(modifier: Modifier = Modifier, permissionsList: List<String>) {
    Column(modifier = modifier) {
        Text(text = "permission list:")
        HorizontalDivider()
        permissionsList.forEach {
            Text(text = it)
            HorizontalDivider()
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
fun StatusScreen(modifier: Modifier = Modifier, dpm: DevicePolicyManager) {
    val tag = "CertStatusScreen"
    Log.i(tag, "this is a log!")

    Greeting(
        name = "banana frosg",
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

suspend fun listKeystoreCertificates(): List<CertificateInfo> = withContext(Dispatchers.IO) {
    try {
        val keyStore = KeyStore.getInstance("AndroidKeyStore")
        keyStore.load(null)

        val aliases = keyStore.aliases().toList()

        aliases.mapNotNull { alias ->
            try {
                val cert = keyStore.getCertificate(alias) as? X509Certificate
                cert?.let {
                    CertificateInfo(
                        alias = alias,
                        subject = it.subjectDN.name,
                        issuer = it.issuerDN.name,
                        notBefore = it.notBefore,
                        notAfter = it.notAfter
                    )
                }
            } catch (e: Exception) {
                null
            }
        }
    } catch (e: Exception) {
        Log.e("Certificate", "Error listing keystore certificates", e)
        emptyList()
    }
}

data class CertificateInfo(
    val alias: String,
    val subject: String,
    val issuer: String,
    val notBefore: Date,
    val notAfter: Date,
)

suspend fun makePostRequest(urlString: String, jsonBody: String): String {
    return withContext(Dispatchers.IO) {
        val tag = "makePostRequest"
        val url = URL(urlString)
        val connection = url.openConnection() as HttpURLConnection

        try {
            connection.requestMethod = "POST"
            connection.connectTimeout = 10000
            connection.readTimeout = 10000
            connection.doOutput = true
            connection.setRequestProperty("Content-Type", "application/json")

            Log.d(tag, "headers set, making request")

            // Write the JSON body
            connection.outputStream.use { os ->
                os.write(jsonBody.toByteArray())
            }

            Log.d(tag, "body written")

            val responseCode = connection.responseCode

            if (responseCode == HttpURLConnection.HTTP_OK) {
                Log.d(tag, "response OK")
                connection.inputStream.bufferedReader().use { it.readText() }
            } else {
                Log.d(tag, "response bad")
                throw Exception("HTTP error code: $responseCode: ${connection.responseMessage}")
            }
        } finally {
            Log.d(tag, "finally")
            connection.disconnect()
        }
    }
}

// Make sure to call this from a coroutine or background thread
suspend fun makeGetRequest(urlString: String): String {
    return withContext(Dispatchers.IO) {
        val tag = "makeGetRequest"
        Log.d(tag, "in withContext")
        Log.d(tag, "url: $urlString")
        val url = URL(urlString)
        val connection = url.openConnection() as HttpURLConnection

        try {
            connection.requestMethod = "GET"
            connection.connectTimeout = 10000 // 10 seconds
            connection.readTimeout = 10000

            val responseCode = connection.responseCode

            if (responseCode == HttpURLConnection.HTTP_OK) {
                connection.inputStream.bufferedReader().use { it.readText() }
            } else {
                throw Exception("HTTP error code: $responseCode")
            }
        } finally {
            connection.disconnect()
        }
    }
}

@Preview(showBackground = true)
@Composable
fun GreetingPreview() {
    MyApplicationTheme {
        Greeting("Android")
    }
}
