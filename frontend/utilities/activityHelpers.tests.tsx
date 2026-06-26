import { getMdmCommandDisplayName } from "./activityHelpers";

describe("getMdmCommandDisplayName function", () => {
  it("returns empty string for undefined", () => {
    expect(getMdmCommandDisplayName(undefined)).toEqual("");
  });

  it("returns empty string for empty string", () => {
    expect(getMdmCommandDisplayName("")).toEqual("");
  });

  it("returns the value as-is for a simple command name with no path separator", () => {
    expect(getMdmCommandDisplayName("DeviceInformation")).toEqual(
      "DeviceInformation"
    );
  });

  it("truncates a multi-segment Windows OMA-URI path to the last segment", () => {
    expect(
      getMdmCommandDisplayName(
        "./Device/Vendor/MSFT/DMClient/Provider/DEMO/EntDMID"
      )
    ).toEqual(".../EntDMID");
  });

  it("handles a trailing slash by ignoring the empty final segment", () => {
    expect(getMdmCommandDisplayName("./Vendor/MSFT/")).toEqual(".../MSFT");
  });
});
