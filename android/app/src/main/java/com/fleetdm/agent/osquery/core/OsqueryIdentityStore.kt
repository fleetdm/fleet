package com.fleetdm.agent.osquery.core

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.first
import java.util.UUID

private val Context.osqueryDataStore by preferencesDataStore(name = "osquery")

object OsqueryIdentityStore {
    private val KEY_UUID = stringPreferencesKey("osquery_uuid")

    suspend fun getOrCreateUuid(context: Context): String {
        val prefs = context.osqueryDataStore.data.first()
        val existing = prefs[KEY_UUID]
        if (!existing.isNullOrBlank()) return existing

        val created = UUID.randomUUID().toString()
        context.osqueryDataStore.edit { it[KEY_UUID] = created }
        return created
    }
}
