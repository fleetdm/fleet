import decodeBase64Utf8 from "./base64";

describe("base64 utilities", () => {
  describe("decodeBase64Utf8 function", () => {
    it("returns an empty string for null, undefined, empty and whitespace-only input", () => {
      // callers pass API fields that are null when the server has nothing to
      // send (e.g. the result of an MDM command that hasn't run yet)
      expect(decodeBase64Utf8(null)).toEqual("");
      expect(decodeBase64Utf8(undefined)).toEqual("");
      expect(decodeBase64Utf8("")).toEqual("");
      expect(decodeBase64Utf8("   ")).toEqual("");
    });

    it("properly decodes UTF-8 base64 strings (including non-ASCII)", () => {
      const utf8 = "Günter Møller Sánchez Peña 🎉";
      const utf8B64 = Buffer.from(utf8, "utf-8").toString("base64");

      const decoded = decodeBase64Utf8(utf8B64);

      expect(decoded).toEqual(utf8);
    });

    it("properly decodes plist XML payloads", () => {
      const xml = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
<key>CommandUUID</key>
<string>REFETCH-CERTS-1a2bc345-678e-90fg</string>
<key>Command</key>
<dict>
<key>ManagedOnly</key>
<false/>
<key>RequestType</key>
<string>CertificateList</string>
</dict>
</dict>
</plist>`;

      const xmlB64 = Buffer.from(xml, "utf-8").toString("base64");

      const decoded = decodeBase64Utf8(xmlB64);

      expect(decoded).toEqual(xml);
      expect(decoded).toContain("CertificateList");
    });
  });
});
