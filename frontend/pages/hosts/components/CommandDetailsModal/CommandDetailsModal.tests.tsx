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
    const { container } = render(
      <ModalContent
        data={{
          results: [
            createCommandResult({ payload: APPLE_PLIST, result: APPLE_PLIST }),
          ],
        }}
        isLoading={false}
        error={null}
      />
    );

    const [payload, result] = container.querySelectorAll("textarea");
    expect(payload).toHaveValue(APPLE_PLIST);
    expect(result).toHaveValue(APPLE_PLIST);
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
});
