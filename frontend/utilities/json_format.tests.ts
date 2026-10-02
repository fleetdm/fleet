import formatJsonForDisplay from "./json_format";

const APPLE_PLIST = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>RequestType</key>
	<string>DeviceInformation</string>
</dict>
</plist>`;

describe("formatJsonForDisplay", () => {
  it("indents minified JSON", () => {
    expect(formatJsonForDisplay('{"done":true,"metadata":{"type":"WIPE"}}'))
      .toEqual(`{
  "done": true,
  "metadata": {
    "type": "WIPE"
  }
}`);
  });

  it("indents a JSON array", () => {
    expect(formatJsonForDisplay("[1,2]")).toEqual(`[
  1,
  2
]`);
  });

  it("leaves an Apple plist untouched", () => {
    expect(formatJsonForDisplay(APPLE_PLIST)).toEqual(APPLE_PLIST);
  });

  it("leaves Windows SyncML untouched", () => {
    const syncML = "<SyncML><SyncBody><Status>200</Status></SyncBody></SyncML>";
    expect(formatJsonForDisplay(syncML)).toEqual(syncML);
  });

  it("leaves invalid JSON untouched", () => {
    expect(formatJsonForDisplay('{"done":true')).toEqual('{"done":true');
  });

  it("leaves a bare scalar untouched", () => {
    expect(formatJsonForDisplay("200")).toEqual("200");
    expect(formatJsonForDisplay('"a string"')).toEqual('"a string"');
    expect(formatJsonForDisplay("null")).toEqual("null");
  });

  it("leaves an empty string untouched", () => {
    expect(formatJsonForDisplay("")).toEqual("");
  });

  it("keeps empty objects and arrays on one line", () => {
    expect(formatJsonForDisplay('{"wipeParams":{},"tags":[]}')).toEqual(`{
  "wipeParams": {},
  "tags": []
}`);
  });

  it("indents a real Android operation without altering any token", () => {
    const operation =
      '{"done":true,"metadata":{"@type":"type.googleapis.com/google.android.devicemanagement.v1.Command","type":"WIPE","createTime":"2026-09-14T18:52:33.942Z"},"name":"enterprises/LC01faqpoi/devices/34d187efe6d96e2d/operations/1789411953942"}';

    expect(formatJsonForDisplay(operation)).toEqual(`{
  "done": true,
  "metadata": {
    "@type": "type.googleapis.com/google.android.devicemanagement.v1.Command",
    "type": "WIPE",
    "createTime": "2026-09-14T18:52:33.942Z"
  },
  "name": "enterprises/LC01faqpoi/devices/34d187efe6d96e2d/operations/1789411953942"
}`);
  });

  // JSON.parse/stringify would corrupt every case below, which is why the
  // formatter indents the raw text instead of reserializing it
  it("preserves integers too large for a JS number", () => {
    expect(formatJsonForDisplay('{"id":9007199254740993}')).toEqual(`{
  "id": 9007199254740993
}`);
  });

  it("preserves key order for integer-like keys", () => {
    expect(formatJsonForDisplay('{"10":"a","2":"b"}')).toEqual(`{
  "10": "a",
  "2": "b"
}`);
  });

  it("preserves number formatting", () => {
    expect(formatJsonForDisplay('{"v":1.0,"e":1e3,"neg":-0}')).toEqual(`{
  "v": 1.0,
  "e": 1e3,
  "neg": -0
}`);
  });

  it("preserves escape sequences, including escaped quotes and braces", () => {
    // Go's json.Marshal escapes "<" as a unicode escape, so stored payloads
    // really do contain escape sequences that must survive formatting
    const withEscapes = '{"msg":"a \\u003c b \\"quoted\\" {not:nested}"}';

    expect(formatJsonForDisplay(withEscapes)).toEqual(`{
  "msg": "a \\u003c b \\"quoted\\" {not:nested}"
}`);
  });

  it("treats an escaped quote as part of the string, not its end", () => {
    // the text after the escaped quote holds a space and a comma, which would
    // be eaten or broken onto a new line if the escape ended the string early
    const escapedQuote = '{"msg":"say \\"hi there\\", ok"}';

    expect(formatJsonForDisplay(escapedQuote)).toEqual(`{
  "msg": "say \\"hi there\\", ok"
}`);
  });

  it("preserves whitespace and separators inside strings", () => {
    expect(formatJsonForDisplay('{"msg":"line one, line two"}')).toEqual(`{
  "msg": "line one, line two"
}`);
  });

  it("reindents JSON that already contains whitespace", () => {
    expect(formatJsonForDisplay('{\n\t"a" : [ 1, 2 ]\n}')).toEqual(`{
  "a": [
    1,
    2
  ]
}`);
  });
});
