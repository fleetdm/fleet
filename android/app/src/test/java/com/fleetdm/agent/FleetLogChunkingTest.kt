package com.fleetdm.agent

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * logcat caps an entry's payload in bytes, so a stack trace has to be split by encoded size --
 * counting characters would still lose the tail of non-ASCII text.
 */
class FleetLogChunkingTest {

    private fun utf8Size(text: String) = text.toByteArray(Charsets.UTF_8).size

    @Test
    fun `text within the limit stays one chunk`() {
        assertEquals(listOf("short message"), chunkForLogcat("short message", 100))
        assertEquals(listOf(""), chunkForLogcat("", 100))
    }

    @Test
    fun `long text is split at line boundaries with nothing lost`() {
        val lines = (1..40).map { "line $it: at com.fleetdm.agent.Something.method(Something.kt:$it)" }
        val text = lines.joinToString("\n")

        val chunks = chunkForLogcat(text, 200)

        assertTrue(chunks.size > 1)
        chunks.forEach { assertTrue(utf8Size(it) <= 200) }
        assertEquals(text, chunks.joinToString("\n"))
    }

    @Test
    fun `chunks respect the byte limit for multibyte text`() {
        // Three bytes per character in UTF-8: a character-counting split would blow the limit.
        val text = "私はテストです".repeat(60)

        val chunks = chunkForLogcat(text, 120)

        assertTrue(chunks.size > 1)
        chunks.forEach { assertTrue("chunk was ${utf8Size(it)} bytes", utf8Size(it) <= 120) }
        assertEquals(text, chunks.joinToString(""))
    }

    @Test
    fun `a single overlong line is split without losing anything`() {
        val text = "a".repeat(500)

        val chunks = chunkForLogcat(text, 100)

        assertEquals(5, chunks.size)
        assertEquals(text, chunks.joinToString(""))
    }

    @Test
    fun `surrogate pairs are never split`() {
        val text = "🔐".repeat(50) // U+1F510, four UTF-8 bytes each

        val chunks = chunkForLogcat(text, 30)

        assertTrue(chunks.size > 1)
        chunks.forEach {
            assertTrue(utf8Size(it) <= 30)
            // A split surrogate pair would not survive an encode/decode round trip.
            assertEquals(it, String(it.toByteArray(Charsets.UTF_8), Charsets.UTF_8))
        }
        assertEquals(text, chunks.joinToString(""))
    }

    @Test
    fun `an empty line at a chunk boundary survives`() {
        val line = "x".repeat(300)
        val text = "$line\n\nnext"

        val chunks = chunkForLogcat(text, 300)

        assertTrue(chunks.size > 1)
        assertEquals(text, chunks.joinToString("\n"))
    }

    @Test
    fun `a leading empty line survives`() {
        val text = "\n" + "a".repeat(90) + "\n" + "b".repeat(90)

        val chunks = chunkForLogcat(text, 100)

        assertTrue(chunks.size > 1)
        assertEquals(text, chunks.joinToString("\n"))
    }

    @Test
    fun `mixed line lengths keep every line intact where they fit`() {
        val text = "short\n" + "x".repeat(250) + "\nshort again"

        val chunks = chunkForLogcat(text, 100)

        chunks.forEach { assertTrue(utf8Size(it) <= 100) }
        assertTrue(chunks.first().startsWith("short"))
        assertTrue(chunks.last().endsWith("short again"))
        assertEquals(text.filterNot { it == '\n' }, chunks.joinToString("").filterNot { it == '\n' })
    }
}
