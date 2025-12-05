package com.fleetdm.agent

import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class KeystoreManagerTest {

    @Test
    fun testEncryptDecrypt() {
        val originalText = "test_api_key_12345"

        val encrypted = KeystoreManager.encrypt(originalText)
        assertNotEquals(originalText, encrypted)

        val decrypted = KeystoreManager.decrypt(encrypted)
        assertEquals(originalText, decrypted)
    }

    @Test
    fun testEncryptProducesDifferentCiphertext() {
        val originalText = "test_api_key_12345"

        val encrypted1 = KeystoreManager.encrypt(originalText)
        val encrypted2 = KeystoreManager.encrypt(originalText)

        assertNotEquals(encrypted1, encrypted2)

        assertEquals(originalText, KeystoreManager.decrypt(encrypted1))
        assertEquals(originalText, KeystoreManager.decrypt(encrypted2))
    }

    @Test(expected = IllegalArgumentException::class)
    fun testDecryptInvalidFormat() {
        KeystoreManager.decrypt("invalid_format")
    }

    @Test
    fun testEncryptEmptyString() {
        val originalText = ""

        val encrypted = KeystoreManager.encrypt(originalText)
        val decrypted = KeystoreManager.decrypt(encrypted)

        assertEquals(originalText, decrypted)
    }

    @Test
    fun testEncryptLongString() {
        val originalText = "a".repeat(10000)

        val encrypted = KeystoreManager.encrypt(originalText)
        val decrypted = KeystoreManager.decrypt(encrypted)

        assertEquals(originalText, decrypted)
    }

    @Test
    fun testEncryptSpecialCharacters() {
        val originalText = "!@#\$%^&*()_+-=[]{}|;':\",./<>?~`"

        val encrypted = KeystoreManager.encrypt(originalText)
        val decrypted = KeystoreManager.decrypt(encrypted)

        assertEquals(originalText, decrypted)
    }
}
