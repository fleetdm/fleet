import decodePlistDataForDisplay from "./plist_format";

const PROFILE = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
    <key>PayloadDisplayName</key>
    <string>👋 Managed bookmarks</string>

    <key>PayloadType</key>
    <string>Configuration</string>
</dict>
</plist>`;

const command = (data: string) => `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>Command</key>
	<dict>
		<key>RequestType</key>
		<string>InstallProfile</string>
		<key>Payload</key>
		<data>${data}</data>
	</dict>
</dict>
</plist>`;

const toBase64 = (bytes: Buffer) => bytes.toString("base64");

// Stands in for the PKCS #7 envelope Fleet signs profiles with: binary DER
// around the profile bytes. 0xe2 right before the XML is a UTF-8 lead byte, so
// it checks that the start of the profile survives a lossy decode.
const signedEnvelope = (profile: string) =>
  Buffer.concat([
    Buffer.from([0x30, 0x82, 0x01, 0x1c, 0x06, 0x09, 0x2a, 0x86, 0x48, 0xe2]),
    Buffer.from(profile, "utf-8"),
    Buffer.from([0x0a, 0xa0, 0x82, 0x03, 0x9b, 0x30, 0x82, 0xff]),
  ]);

describe("decodePlistDataForDisplay", () => {
  it("shows the profile inside a signed InstallProfile payload, indented under <data>", () => {
    const b64 = toBase64(signedEnvelope(PROFILE));

    expect(decodePlistDataForDisplay(command(b64))).toEqual(
      command(`
			<?xml version="1.0" encoding="UTF-8"?>
			<plist version="1.0">
			<dict>
			    <key>PayloadDisplayName</key>
			    <string>👋 Managed bookmarks</string>

			    <key>PayloadType</key>
			    <string>Configuration</string>
			</dict>
			</plist>
		`)
    );
  });

  it("shows an unsigned plist payload", () => {
    const b64 = toBase64(Buffer.from(`${PROFILE}\n`, "utf-8"));

    expect(decodePlistDataForDisplay(command(b64))).toContain(
      "\t\t\t    <string>👋 Managed bookmarks</string>"
    );
  });

  it("decodes base64 that is wrapped across lines", () => {
    const b64 = toBase64(signedEnvelope(PROFILE)).replace(
      /(.{60})/g,
      "$1\n\t\t"
    );

    expect(decodePlistDataForDisplay(command(b64))).toContain(
      "\t\t\t    <string>Configuration</string>"
    );
  });

  it("keeps single-line text inline", () => {
    const tokens = '{"SyncTokens":{"Timestamp":"2026-10-14T00:00:00Z"}}';
    const b64 = toBase64(Buffer.from(tokens, "utf-8"));

    expect(decodePlistDataForDisplay(command(b64))).toEqual(command(tokens));
  });

  it("leaves binary data like a certificate as base64", () => {
    const cert = toBase64(
      Buffer.from([0x30, 0x82, 0x03, 0x97, 0x30, 0x82, 0x02, 0x7f, 0xa0, 0x03])
    );

    expect(decodePlistDataForDisplay(command(cert))).toEqual(command(cert));
  });

  it("leaves binary with an unterminated plist as base64", () => {
    const b64 = toBase64(
      Buffer.concat([
        Buffer.from([0x30, 0x82]),
        Buffer.from('<plist version="1.0"><dict>', "utf-8"),
        Buffer.from([0xff]),
      ])
    );

    expect(decodePlistDataForDisplay(command(b64))).toEqual(command(b64));
  });

  it("leaves an empty <data> untouched", () => {
    expect(decodePlistDataForDisplay(command(""))).toEqual(command(""));
  });

  it("decodes each <data> on its own, indenting under the line it's on", () => {
    const cert = toBase64(Buffer.from([0x30, 0x82, 0x03, 0x97]));
    const text = toBase64(Buffer.from("first\r\nsecond", "utf-8"));
    const plist = `<dict>
	<key>Cert</key><data>${cert}</data>
	<key>Text</key><data>${text}</data>
</dict>`;

    expect(decodePlistDataForDisplay(plist)).toEqual(`<dict>
	<key>Cert</key><data>${cert}</data>
	<key>Text</key><data>
		first
		second
	</data>
</dict>`);
  });

  it("leaves invalid base64 untouched", () => {
    expect(decodePlistDataForDisplay(command("not base64!"))).toEqual(
      command("not base64!")
    );
  });

  it("leaves a plist without <data> untouched", () => {
    const plist = command("").replace("<data></data>", "<string>x</string>");

    expect(decodePlistDataForDisplay(plist)).toEqual(plist);
  });

  it("leaves Windows SyncML untouched", () => {
    const syncML = "<SyncML><Item><Data>MTIz</Data></Item></SyncML>";

    expect(decodePlistDataForDisplay(syncML)).toEqual(syncML);
  });
});
