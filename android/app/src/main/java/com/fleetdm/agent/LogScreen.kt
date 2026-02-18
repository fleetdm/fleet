package com.fleetdm.agent

import android.content.ClipData
import android.content.ClipboardManager
import android.widget.Toast
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

private enum class LogLevel(val label: String, val logcatCode: String) {
    VERBOSE("Verbose", "V"),
    DEBUG("Debug", "D"),
    INFO("Info", "I"),
    WARNING("Warning", "W"),
    ERROR("Error", "E"),
}

@Composable
fun LogsScreen(onNavigateBack: () -> Unit) {
    var logs by remember { mutableStateOf<String?>(null) }
    var selectedLevel by remember { mutableStateOf(LogLevel.INFO) }
    var levelMenuExpanded by remember { mutableStateOf(false) }
    val context = LocalContext.current
    val scrollState = rememberScrollState()

    LaunchedEffect(logs) {
        if (logs != null) scrollState.scrollTo(scrollState.maxValue)
    }

    LaunchedEffect(selectedLevel) {
        logs = null
        logs = withContext(Dispatchers.IO) {
            try {
                val fleetTags = listOf(
                    "fleet-app", "fleet-ApiClient", "fleet-CertificateEnrollmentWorker",
                    "fleet-CertificateOrchestrator", "fleet-AndroidCertInstaller",
                    "fleet-DeviceKeystoreManager", "fleet-boot", "fleet-RoleNotificationReceiverService",
                )
                val filterArgs = fleetTags.map { "$it:${selectedLevel.logcatCode}" } + listOf("*:S")
                val uid = context.applicationInfo.uid
                val command = listOf("logcat", "-d") + filterArgs
                val process = ProcessBuilder(command).redirectErrorStream(true).start()
                process.inputStream.bufferedReader().readText()
            } catch (e: Exception) {
                "Failed to read logs: ${e.message}"
            }
        }
    }

    Scaffold(
        modifier = Modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text("Logs") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    Box {
                        TextButton(onClick = { levelMenuExpanded = true }) {
                            Text(selectedLevel.label)
                        }
                        DropdownMenu(
                            expanded = levelMenuExpanded,
                            onDismissRequest = { levelMenuExpanded = false },
                        ) {
                            LogLevel.entries.forEach { level ->
                                DropdownMenuItem(
                                    text = { Text(level.label) },
                                    onClick = {
                                        selectedLevel = level
                                        levelMenuExpanded = false
                                    },
                                )
                            }
                        }
                    }
                    IconButton(onClick = {
                        val clipboard = context.getSystemService(ClipboardManager::class.java)
                            ?: return@IconButton
                        clipboard.setPrimaryClip(ClipData.newPlainText("fleet logs", logs ?: ""))
                        Toast.makeText(context, "Logs copied", Toast.LENGTH_SHORT).show()
                    }) {
                        Icon(Icons.Filled.ContentCopy, contentDescription = "copy logs")
                    }
                },
            )
        },
        content = { paddingValues ->
            if (logs == null) {
                Box(Modifier.fillMaxSize().padding(paddingValues), contentAlignment = Alignment.Center) {
                    Text("Loading logs…")
                }
            } else {
                Text(
                    text = logs!!,
                    modifier = Modifier
                        .padding(paddingValues)
                        .padding(horizontal = 12.dp)
                        .verticalScroll(scrollState),
                    fontFamily = FontFamily.Monospace,
                    fontSize = 10.sp,
                    lineHeight = 12.sp,
                )
            }
        },
    )
}
