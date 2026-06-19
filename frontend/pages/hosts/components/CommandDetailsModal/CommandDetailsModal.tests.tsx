import { GetIconName, getVerbForCommandStatus } from "./CommandDetailsModal";

describe("GetIconName", () => {
  it("returns error for Apple Error status", () => {
    expect(GetIconName("Error")).toEqual("error");
  });

  it("returns error for Apple CommandFormatError status", () => {
    expect(GetIconName("CommandFormatError")).toEqual("error");
  });

  it("returns success for Apple Acknowledged status", () => {
    expect(GetIconName("Acknowledged")).toEqual("success");
  });

  it("returns pending-outline for Apple Pending status", () => {
    expect(GetIconName("Pending")).toEqual("pending-outline");
  });

  it("returns pending-outline for Apple NotNow status", () => {
    expect(GetIconName("NotNow")).toEqual("pending-outline");
  });

  it("returns success for Windows 200 status", () => {
    expect(GetIconName("200")).toEqual("success");
  });

  it("returns error for Windows 400 status", () => {
    expect(GetIconName("400")).toEqual("error");
  });

  it("returns error for Windows 500 status", () => {
    expect(GetIconName("500")).toEqual("error");
  });

  it("returns pending-outline for Windows 101 status", () => {
    expect(GetIconName("101")).toEqual("pending-outline");
  });

  it("returns pending-outline for Windows 199 status (upper pending boundary)", () => {
    expect(GetIconName("199")).toEqual("pending-outline");
  });

  it("returns success for Windows 399 status (upper success boundary)", () => {
    expect(GetIconName("399")).toEqual("success");
  });

  it("returns warning for an unknown status", () => {
    expect(GetIconName("unknown")).toEqual("warning");
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
