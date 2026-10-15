import { render, screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import { createMockAppleMdmCommandResult } from "__mocks__/commandMock";
import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import CommandResultsModal, {
  decodeCommandResults,
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

describe("decodeCommandResults", () => {
  it("decodes a null result to an empty string rather than garbage", () => {
    // the API returns a null result for a command that hasn't run yet (e.g. a
    // pending Android command). atob(null) stringifies null to "null", which is
    // valid base64 and decodes to a truthy "\x9eée", faking a device response.
    const decoded = decodeCommandResults({
      results: [createMockAppleMdmCommandResult({ result: null })],
    });

    expect(decoded.results?.[0].result).toEqual("");
  });

  it("decodes a null payload to an empty string rather than garbage", () => {
    const decoded = decodeCommandResults({
      results: [createMockAppleMdmCommandResult({ payload: null })],
    });

    expect(decoded.results?.[0].payload).toEqual("");
  });

  it("decodes an empty-string result to an empty string", () => {
    const decoded = decodeCommandResults({
      results: [createMockAppleMdmCommandResult({ result: "" })],
    });

    expect(decoded.results?.[0].result).toEqual("");
  });

  it("decodes a result containing multi-byte UTF-8 without mangling it", () => {
    const message = "Perdu ? Téléphonez au +33 1 23 45 67 89 🙏";
    const decoded = decodeCommandResults({
      results: [
        createMockAppleMdmCommandResult({
          result: Buffer.from(message, "utf-8").toString("base64"),
        }),
      ],
    });

    expect(decoded.results?.[0].result).toEqual(message);
  });

  it("decodes an unparseable result to an empty string rather than throwing", () => {
    const consoleError = jest.spyOn(console, "error").mockImplementation();

    const decoded = decodeCommandResults({
      results: [createMockAppleMdmCommandResult({ result: "not base64!!" })],
    });

    expect(decoded.results?.[0].result).toEqual("");
    expect(consoleError).toHaveBeenCalled();

    consoleError.mockRestore();
  });

  it("passes through a response with no results key, which is what the API sends when there is nothing to return", () => {
    expect(decodeCommandResults({})).toEqual({});
  });
});

describe("ModalContent", () => {
  it("renders normally, not as an error, when the API returns a 200 with no results (e.g. host re-enrolled since the command was sent)", () => {
    render(<ModalContent data={{}} isLoading={false} error={null} />);

    expect(
      screen.getByText("This command has been deleted.")
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/something's gone wrong/i)
    ).not.toBeInTheDocument();
  });

  it("does not render a response section for a pending command that has no result", () => {
    render(
      <ModalContent
        data={decodeCommandResults({
          results: [
            createMockAppleMdmCommandResult({
              status: "Pending",
              request_type: "REQUEST_DEVICE_INFO",
              hostname: "Samsung SM-S906U1",
              result: null,
            }),
          ],
        })}
        isLoading={false}
        error={null}
      />
    );

    expect(screen.queryByText(/Response from/i)).not.toBeInTheDocument();
  });

  it("renders the response section for a command that has a result", () => {
    render(
      <ModalContent
        data={decodeCommandResults({
          results: [
            createMockAppleMdmCommandResult({
              hostname: "Samsung SM-S906U1",
              result: btoa("Device is unlocked"),
            }),
          ],
        })}
        isLoading={false}
        error={null}
      />
    );

    expect(screen.getByText(/Response from/i)).toBeInTheDocument();
    expect(screen.getByDisplayValue("Device is unlocked")).toBeInTheDocument();
  });

  it("pretty-prints minified Android JSON in both boxes", () => {
    const { container } = render(
      <ModalContent
        data={decodeCommandResults({
          results: [
            createMockAppleMdmCommandResult({
              request_type: "WIPE",
              hostname: "Samsung SM-S906U1",
              payload: btoa('{"type":"WIPE","wipeParams":{}}'),
              result: btoa('{"done":true,"response":{"errorCode":"NONE"}}'),
            }),
          ],
        })}
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
        data={decodeCommandResults({
          results: [
            createMockAppleMdmCommandResult({
              payload: btoa(APPLE_PLIST),
              result: btoa(resultPlist),
            }),
          ],
        })}
        isLoading={false}
        error={null}
      />
    );

    const [payload, result] = container.querySelectorAll("textarea");
    expect(payload).toHaveValue(APPLE_PLIST);
    expect(result).toHaveValue(resultPlist);
  });
});

describe("CommandResultsModal", () => {
  const renderModal = createCustomRenderer({ withBackendMock: true });

  it("does not render a fabricated response for a pending command the API returns a null result for", async () => {
    mockServer.use(
      http.get(baseUrl("/commands/results"), () =>
        HttpResponse.json({
          results: [
            {
              host_uuid: "11111111-2222-3333-4444-555555555555",
              command_uuid: "pending-android-command",
              status: "Pending",
              updated_at: "2025-08-10T12:05:00Z",
              request_type: "REQUEST_DEVICE_INFO",
              hostname: "Samsung SM-S906U1",
              payload: btoa('{"type":"REQUEST_DEVICE_INFO"}'),
              result: null,
              name: null,
            },
          ],
        })
      )
    );

    renderModal(
      <CommandResultsModal
        command={{ command_uuid: "pending-android-command" }}
        onDone={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(screen.getByText(/is pending on/i)).toBeInTheDocument();
    });

    expect(screen.queryByText(/Response from/i)).not.toBeInTheDocument();
    // atob(null) decodes to this, the exact garbage reported in #53155
    expect(screen.queryByDisplayValue(/ée/)).not.toBeInTheDocument();
  });

  it("renders the response for a command the API returns a result for", async () => {
    mockServer.use(
      http.get(baseUrl("/commands/results"), () =>
        HttpResponse.json({
          results: [
            {
              host_uuid: "11111111-2222-3333-4444-555555555555",
              command_uuid: "completed-android-command",
              status: "Acknowledged",
              updated_at: "2025-08-10T12:05:00Z",
              request_type: "REQUEST_DEVICE_INFO",
              hostname: "Samsung SM-S906U1",
              payload: btoa('{"type":"REQUEST_DEVICE_INFO"}'),
              result: btoa('{"eid":"89049032"}'),
              name: null,
            },
          ],
        })
      )
    );

    renderModal(
      <CommandResultsModal
        command={{ command_uuid: "completed-android-command" }}
        onDone={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(screen.getByText(/Response from/i)).toBeInTheDocument();
    });

    // getByDisplayValue collapses whitespace, so assert on the element itself
    const [, result] = document.querySelectorAll("textarea");
    expect(result).toHaveValue(`{
  "eid": "89049032"
}`);
  });
});
