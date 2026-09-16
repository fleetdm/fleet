import { render, screen } from "@testing-library/react";
import React from "react";

import { ICommandResult } from "interfaces/command";

import {
  formatCommandJson,
  getIconName,
  getVerbForCommandStatus,
  ModalContent,
} from "./CommandDetailsModal";

const APPLE_PLIST = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>RequestType</key>
	<string>DeviceInformation</string>
</dict>
</plist>`;

const createCommandResult = (
  overrides: Partial<ICommandResult> = {}
): ICommandResult => ({
  host_uuid: "host-uuid",
  command_uuid: "command-uuid",
  status: "Acknowledged",
  updated_at: "2026-09-14T18:52:33Z",
  request_type: "WIPE",
  hostname: "test-host",
  payload: "",
  result: "",
  name: null,
  ...overrides,
});

describe("getIconName", () => {
  it("returns error for Apple Error status", () => {
    expect(getIconName("Error")).toEqual("error");
  });

  it("returns error for Apple CommandFormatError status", () => {
    expect(getIconName("CommandFormatError")).toEqual("error");
  });

  it("returns success for Apple Acknowledged status", () => {
    expect(getIconName("Acknowledged")).toEqual("success");
  });

  it("returns pending-outline for Apple Pending status", () => {
    expect(getIconName("Pending")).toEqual("pending-outline");
  });

  it("returns pending-outline for Apple NotNow status", () => {
    expect(getIconName("NotNow")).toEqual("pending-outline");
  });

  it("returns success for Windows 200 status", () => {
    expect(getIconName("200")).toEqual("success");
  });

  it("returns error for Windows 400 status", () => {
    expect(getIconName("400")).toEqual("error");
  });

  it("returns error for Windows 500 status", () => {
    expect(getIconName("500")).toEqual("error");
  });

  it("returns pending-outline for Windows 101 status", () => {
    expect(getIconName("101")).toEqual("pending-outline");
  });

  it("returns pending-outline for Windows 199 status (upper pending boundary)", () => {
    expect(getIconName("199")).toEqual("pending-outline");
  });

  it("returns success for Windows 399 status (upper success boundary)", () => {
    expect(getIconName("399")).toEqual("success");
  });

  it("returns warning for an unknown status", () => {
    expect(getIconName("unknown")).toEqual("warning");
  });
});

describe("getVerbForCommandStatus", () => {
  it("returns 'ran' for a successful status", () => {
    expect(getVerbForCommandStatus("Acknowledged")).toEqual("ran");
  });

  it("returns 'failed to run' for an error status", () => {
    expect(getVerbForCommandStatus("Error")).toEqual("failed to run");
  });

  it("returns 'sent' for a pending status", () => {
    expect(getVerbForCommandStatus("Pending")).toEqual("sent");
  });

  it("returns 'sent' for an unknown status", () => {
    expect(getVerbForCommandStatus("unknown")).toEqual("sent");
  });
});

describe("ModalContent", () => {
  it("renders normally, not as an error, when the API returns a 200 with no results (e.g. host re-enrolled since the command was sent)", () => {
    render(
      <ModalContent data={{ results: [] }} isLoading={false} error={null} />
    );

    expect(
      screen.getByText("This command has been deleted.")
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/something's gone wrong/i)
    ).not.toBeInTheDocument();
  });

  it("pretty-prints minified Android JSON in both boxes", () => {
    const { container } = render(
      <ModalContent
        data={{
          results: [
            createCommandResult({
              payload: '{"type":"WIPE","wipeParams":{}}',
              result: '{"done":true,"response":{"errorCode":"NONE"}}',
            }),
          ],
        }}
        isLoading={false}
        error={null}
      />
    );

    const [payload, result] = container.querySelectorAll("textarea");
    expect(payload).toHaveValue(`{
  "type": "WIPE",
  "wipeParams": {}
}`);
    expect(result).toHaveValue(`{
  "done": true,
  "response": {
    "errorCode": "NONE"
  }
}`);
  });

  it("renders an Apple plist payload and result unchanged", () => {
    const resultPlist = APPLE_PLIST.replace(
      "DeviceInformation",
      "DeviceInformationResponse"
    );
    const { container } = render(
      <ModalContent
        data={{
          results: [
            createCommandResult({ payload: APPLE_PLIST, result: resultPlist }),
          ],
        }}
        isLoading={false}
        error={null}
      />
    );

    const [payload, result] = container.querySelectorAll("textarea");
    expect(payload).toHaveValue(APPLE_PLIST);
    expect(result).toHaveValue(resultPlist);
  });

  it("renders only the payload box for a command with no result yet", () => {
    const { container } = render(
      <ModalContent
        data={{
          results: [
            createCommandResult({
              status: "Pending",
              payload: '{"type":"WIPE"}',
              result: "",
            }),
          ],
        }}
        isLoading={false}
        error={null}
      />
    );

    const textareas = container.querySelectorAll("textarea");
    expect(textareas).toHaveLength(1);
    expect(textareas[0]).toHaveValue(`{
  "type": "WIPE"
}`);
    expect(screen.queryByText(/Response from/)).not.toBeInTheDocument();
  });
});

describe("formatCommandJson", () => {
  it("indents minified JSON", () => {
    expect(formatCommandJson('{"done":true,"metadata":{"type":"WIPE"}}'))
      .toEqual(`{
  "done": true,
  "metadata": {
    "type": "WIPE"
  }
}`);
  });

  it("indents a JSON array", () => {
    expect(formatCommandJson("[1,2]")).toEqual(`[
  1,
  2
]`);
  });

  it("leaves an Apple plist untouched", () => {
    expect(formatCommandJson(APPLE_PLIST)).toEqual(APPLE_PLIST);
  });

  it("leaves Windows SyncML untouched", () => {
    const syncML = "<SyncML><SyncBody><Status>200</Status></SyncBody></SyncML>";
    expect(formatCommandJson(syncML)).toEqual(syncML);
  });

  it("leaves invalid JSON untouched", () => {
    expect(formatCommandJson('{"done":true')).toEqual('{"done":true');
  });

  it("leaves a bare scalar untouched", () => {
    expect(formatCommandJson("200")).toEqual("200");
    expect(formatCommandJson('"a string"')).toEqual('"a string"');
    expect(formatCommandJson("null")).toEqual("null");
  });

  it("leaves an empty string untouched", () => {
    expect(formatCommandJson("")).toEqual("");
  });

  it("keeps empty objects and arrays on one line", () => {
    expect(formatCommandJson('{"wipeParams":{},"tags":[]}')).toEqual(`{
  "wipeParams": {},
  "tags": []
}`);
  });

  it("indents a real Android operation without altering any token", () => {
    const operation =
      '{"done":true,"metadata":{"@type":"type.googleapis.com/google.android.devicemanagement.v1.Command","type":"WIPE","createTime":"2026-09-14T18:52:33.942Z"},"name":"enterprises/LC01faqpoi/devices/34d187efe6d96e2d/operations/1789411953942"}';

    expect(formatCommandJson(operation)).toEqual(`{
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
    expect(formatCommandJson('{"id":9007199254740993}')).toEqual(`{
  "id": 9007199254740993
}`);
  });

  it("preserves key order for integer-like keys", () => {
    expect(formatCommandJson('{"10":"a","2":"b"}')).toEqual(`{
  "10": "a",
  "2": "b"
}`);
  });

  it("preserves number formatting", () => {
    expect(formatCommandJson('{"v":1.0,"e":1e3,"neg":-0}')).toEqual(`{
  "v": 1.0,
  "e": 1e3,
  "neg": -0
}`);
  });

  it("preserves escape sequences, including escaped quotes and braces", () => {
    // Go's json.Marshal escapes "<" as a unicode escape, so stored payloads
    // really do contain escape sequences that must survive formatting
    const withEscapes = '{"msg":"a \\u003c b \\"quoted\\" {not:nested}"}';

    expect(formatCommandJson(withEscapes)).toEqual(`{
  "msg": "a \\u003c b \\"quoted\\" {not:nested}"
}`);
  });

  it("treats an escaped quote as part of the string, not its end", () => {
    // the text after the escaped quote holds a space and a comma, which would
    // be eaten or broken onto a new line if the escape ended the string early
    const escapedQuote = '{"msg":"say \\"hi there\\", ok"}';

    expect(formatCommandJson(escapedQuote)).toEqual(`{
  "msg": "say \\"hi there\\", ok"
}`);
  });

  it("preserves whitespace and separators inside strings", () => {
    expect(formatCommandJson('{"msg":"line one, line two"}')).toEqual(`{
  "msg": "line one, line two"
}`);
  });

  it("reindents JSON that already contains whitespace", () => {
    expect(formatCommandJson('{\n\t"a" : [ 1, 2 ]\n}')).toEqual(`{
  "a": [
    1,
    2
  ]
}`);
  });
});
