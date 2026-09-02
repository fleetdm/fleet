import { ISetupStep } from "interfaces/setup";
import {
  canAutoInitiateDeviceSSO,
  clearDeviceSSOAttempt,
  isSoftwareScriptSetup,
  isSSORequiredError,
  recordDeviceSSOAttempt,
} from "./helpers";

const setupStep = (source?: ISetupStep["source"]): ISetupStep => ({
  name: "test",
  status: "success",
  type: "software_script_run",
  source,
});

describe("isSoftwareScriptSetup", () => {
  it("returns true for script package sources (sh, ps1, py)", () => {
    expect(isSoftwareScriptSetup(setupStep("sh_packages"))).toBe(true);
    expect(isSoftwareScriptSetup(setupStep("ps1_packages"))).toBe(true);
    expect(isSoftwareScriptSetup(setupStep("py_packages"))).toBe(true);
  });

  it("returns false for non-script sources", () => {
    expect(isSoftwareScriptSetup(setupStep("apps"))).toBe(false);
  });

  it("returns false when source is missing", () => {
    expect(isSoftwareScriptSetup(setupStep(undefined))).toBe(false);
  });
});

describe("isSSORequiredError", () => {
  it("recognizes a 401 carrying the sso_required marker", () => {
    expect(
      isSSORequiredError({ status: 401, data: { sso_required: true } })
    ).toBe(true);
  });

  it("rejects a 401 without the marker, which means a stale device token", () => {
    expect(isSSORequiredError({ status: 401, data: {} })).toBe(false);
    expect(isSSORequiredError({ status: 401 })).toBe(false);
    expect(
      isSSORequiredError({ status: 401, data: { sso_required: false } })
    ).toBe(false);
  });

  it("rejects the marker on any status other than 401", () => {
    expect(
      isSSORequiredError({ status: 500, data: { sso_required: true } })
    ).toBe(false);
  });

  it("tolerates the shapes a failed request can reject with", () => {
    expect(isSSORequiredError(undefined)).toBe(false);
    expect(isSSORequiredError(null)).toBe(false);
    expect(isSSORequiredError(new Error("Network Error"))).toBe(false);
    expect(isSSORequiredError("send request: parse server error")).toBe(false);
  });
});

describe("device SSO attempt flag", () => {
  beforeEach(() => {
    window.sessionStorage.clear();
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("records and clears an attempt for one token at a time", () => {
    expect(canAutoInitiateDeviceSSO("token-a")).toBe(true);

    expect(recordDeviceSSOAttempt("token-a")).toBe(true);
    expect(canAutoInitiateDeviceSSO("token-a")).toBe(false);
    expect(canAutoInitiateDeviceSSO("token-b")).toBe(true);

    expect(clearDeviceSSOAttempt("token-a")).toBe(true);
    expect(canAutoInitiateDeviceSSO("token-a")).toBe(true);
  });

  it("suppresses automatic attempts, rather than granting fresh ones, when storage throws", () => {
    (["getItem", "setItem", "removeItem"] as const).forEach((method) => {
      jest.spyOn(Storage.prototype, method).mockImplementation(() => {
        throw new Error("blocked");
      });
    });

    expect(recordDeviceSSOAttempt("token-a")).toBe(false);
    expect(clearDeviceSSOAttempt("token-a")).toBe(false);
    expect(canAutoInitiateDeviceSSO("token-a")).toBe(false);
  });

  it("reports a write that doesn't stick as not recorded", () => {
    jest
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => undefined);

    expect(recordDeviceSSOAttempt("token-a")).toBe(false);
  });
});
