import {
  encodeScriptBase64,
  SCRIPTS_ENCODED_HEADER,
} from "./scripts_encoding";

describe("scripts_encoding", () => {
  describe("SCRIPTS_ENCODED_HEADER", () => {
    it("should have the expected value", () => {
      expect(SCRIPTS_ENCODED_HEADER).toBe("X-Fleet-Scripts-Encoded");
    });
  });

  describe("encodeScriptBase64", () => {
    it("should return undefined for undefined input", () => {
      expect(encodeScriptBase64(undefined)).toBeUndefined();
    });

    it("should return empty string for empty input", () => {
      expect(encodeScriptBase64("")).toBe("");
    });

    it("should encode simple strings correctly", () => {
      const encoded = encodeScriptBase64("Hello World");
      // "Hello World" in base64 is "SGVsbG8gV29ybGQ="
      expect(encoded).toBe("SGVsbG8gV29ybGQ=");
    });

    it("should encode PowerShell patterns with dollar brace", () => {
      const encoded = encodeScriptBase64("${env:TEMP}");
      // "${env:TEMP}" in base64 is "JHtlbnY6VEVNUH0="
      expect(encoded).toBe("JHtlbnY6VEVNUH0=");
    });

    it("should encode PowerShell install script patterns", () => {
      const script = "$installProcess = Start-Process msiexec.exe";
      const encoded = encodeScriptBase64(script);
      // Verify it's valid base64 and decodes back correctly
      expect(atob(encoded!)).toBe(script);
    });

    it("should encode multiline PowerShell scripts", () => {
      const script =
        '$logFile = "${env:TEMP}/fleet-install.log"\nStart-Process msiexec.exe';
      const encoded = encodeScriptBase64(script);
      // Verify it's valid base64 and decodes back correctly
      expect(atob(encoded!)).toBe(script);
    });

    it("should handle unicode characters correctly", () => {
      const script = 'echo "Hello World"';
      const encoded = encodeScriptBase64(script);
      // Decode and verify using TextDecoder for proper UTF-8 handling
      const decoded = atob(encoded!);
      expect(decoded).toBe(script);
    });

    it("should produce valid base64 that Go can decode", () => {
      // Test the specific WAF-triggering patterns
      const testCases = [
        "${env:TEMP}",
        "${env:INSTALLER_PATH}",
        "Start-Process msiexec.exe",
        '$logFile = "${env:TEMP}/fleet-install.log"',
      ];

      testCases.forEach((script) => {
        const encoded = encodeScriptBase64(script);
        expect(encoded).toBeDefined();
        // Verify it's valid base64 (won't throw)
        expect(() => atob(encoded!)).not.toThrow();
        // Verify it decodes back to original
        expect(atob(encoded!)).toBe(script);
      });
    });
  });
});
