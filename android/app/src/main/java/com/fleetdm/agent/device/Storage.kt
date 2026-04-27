package com.fleetdm.agent.device

import android.content.Context

object Storage {

    private lateinit var context: Context
    private const val FILE = "fleet_device_id"

    fun init(ctx: Context) {
        context = ctx.applicationContext
    }

    fun write(value: String) {
        context.openFileOutput(FILE, Context.MODE_PRIVATE).use {
            it.write(value.toByteArray())
        }
    }

    fun read(): String? {
        return try {
            context.openFileInput(FILE)
                .bufferedReader()
                .use { it.readText() }
        } catch (_: Exception) {
            null
        }
    }
}
