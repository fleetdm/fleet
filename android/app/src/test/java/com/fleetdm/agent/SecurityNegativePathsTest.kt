package com.fleetdm.agent

import android.content.Context
import androidx.datastore.preferences.core.edit
import com.fleetdm.agent.osquery.OsqueryQueryEngine
import com.fleetdm.agent.osquery.OsqueryTables
import kotlinx.coroutines.test.runTest
import org.junit.After
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment

@RunWith(RobolectricTestRunner::class)
class SecurityNegativePathsTest {

    private lateinit var context: Context

    @Before
    fun setup() = runTest {
        KeystoreManager.enableTestMode()
        context = RuntimeEnvironment.getApplication()
        ApiClient.initialize(context)
        context.prefDataStore.edit { it.clear() }
        OsqueryTables.registerAll(context)
    }

    @After
    fun tearDown() {
        KeystoreManager.disableTestMode()
    }

    @Test
    fun `missing enrollment config fails closed`() = runTest {
        val result = ApiClient.getOrbitConfig()
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("Credentials not set") == true)
    }

    @Test
    fun `malformed SQL is rejected`() = runTest {
        val result = runCatching {
            OsqueryQueryEngine.execute("SELECT FROM os_version")
        }
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("Bad SQL") == true)
    }
}
