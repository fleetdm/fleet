package com.fleetdm.agent

import android.content.Context
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * The vitals the agent reports to PUT /api/fleetd/vitals, for what AMAPI doesn't expose.
 *
 * Every field is optional: a null field is left out of the request and the server keeps its previous value.
 * To add a vital, add a nullable field here and read it in [DeviceVitals.collect].
 */
@Serializable
data class DeviceVitals(
    @SerialName("roaming")
    val roaming: Boolean? = null,
) {
    fun isEmpty(): Boolean = this == DeviceVitals()

    companion object {
        fun collect(context: Context): DeviceVitals = DeviceVitals(
            roaming = RoamingReader.read(context),
        )
    }
}
