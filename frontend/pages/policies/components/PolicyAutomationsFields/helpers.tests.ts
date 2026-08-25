import { createMockMdmProfile } from "__mocks__/mdmMock";

import { filterValidProfiles, rewriteProfilePlatform } from "./helpers";

describe("filterValidProfiles", () => {
  it("keeps macOS and Windows configuration profiles", () => {
    expect(
      filterValidProfiles(
        createMockMdmProfile({ platform: "darwin", profile_uuid: "a-123" })
      )
    ).toBe(true);
    expect(
      filterValidProfiles(
        createMockMdmProfile({ platform: "windows", profile_uuid: "w-123" })
      )
    ).toBe(true);
  });

  it("excludes Apple declarations, which share the darwin platform", () => {
    expect(
      filterValidProfiles(
        createMockMdmProfile({ platform: "darwin", profile_uuid: "d-123" })
      )
    ).toBe(false);
  });

  it("excludes platforms that don't support policies", () => {
    expect(
      filterValidProfiles(
        createMockMdmProfile({ platform: "ios", profile_uuid: "a-123" })
      )
    ).toBe(false);
  });
});

describe("rewriteProfilePlatform", () => {
  it("renders supported platforms with their display names", () => {
    expect(rewriteProfilePlatform("darwin")).toBe("macOS");
    expect(rewriteProfilePlatform("windows")).toBe("Windows");
  });

  it("falls back for unsupported platforms", () => {
    expect(rewriteProfilePlatform("android")).toBe("Unsupported");
  });
});
