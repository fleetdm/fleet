import { ISetupStep } from "interfaces/setup";
import { isSoftwareScriptSetup } from "./helpers";

const setupStep = (source?: ISetupStep["source"]): ISetupStep => ({
  name: "test",
  status: "success",
  type: "software_script_run",
  source,
});

describe("DeviceUserPage helpers - isSoftwareScriptSetup", () => {
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
