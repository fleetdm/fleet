package com.fleetdm.agent

import android.content.Context
import androidx.work.ListenableWorker
import androidx.work.testing.TestListenableWorkerBuilder
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import kotlinx.coroutines.runBlocking

@RunWith(RobolectricTestRunner::class)
class ConfigCheckWorkerTest {
    private lateinit var context: Context

    @Before
    fun setUp() {
        context = RuntimeEnvironment.getApplication()
    }

    @Test
    fun testDoWork() {
        val worker =
            TestListenableWorkerBuilder<ConfigCheckWorker>(context)
                .build()

        // Execute the worker (doWork is now a suspend function)
        val result = runBlocking { worker.doWork() }
        assertEquals(ListenableWorker.Result.success(), result)
    }
}
