package com.fleetdm.agent.osquery

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.first

private val Context.dataStore by preferencesDataStore(name = "fleet_prefs")

object FleetPreferences {

    private val NODE_KEY = stringPreferencesKey("fleet_node_key")

    suspend fun getNodeKey(context: Context): String? {
        return context.dataStore.data.first()[NODE_KEY]
    }

    suspend fun setNodeKey(context: Context, nodeKey: String) {
        context.dataStore.edit { prefs ->
            prefs[NODE_KEY] = nodeKey
        }
    }

    suspend fun clearNodeKey(context: Context) {
        context.dataStore.edit { prefs ->
            prefs.remove(NODE_KEY)
        }
    }
}

