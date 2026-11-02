import { createMockVppApp } from "__mocks__/appleMdm";
import sendRequest from "services";

import softwareAPI from "./software";

jest.mock("services", () => ({
  __esModule: true,
  default: jest.fn(),
}));

const mockSendRequest = sendRequest as jest.MockedFunction<typeof sendRequest>;

describe("softwareAPI.getSoftwarePackageToken", () => {
  beforeEach(() => {
    mockSendRequest.mockReset();
    mockSendRequest.mockResolvedValue({ token: "test-token" });
  });

  it("omits installer_id from the query when undefined (single-package back-compat)", async () => {
    await softwareAPI.getSoftwarePackageToken(42, 7);

    const [method, path] = mockSendRequest.mock.calls[0];
    expect(method).toBe("POST");
    expect(path).toContain("/software/titles/42/package/token");
    expect(path).toContain("alt=media");
    expect(path).toContain("fleet_id=7");
    expect(path).not.toContain("installer_id");
  });

  it("forwards installer_id as a query param when provided (#49239)", async () => {
    await softwareAPI.getSoftwarePackageToken(42, 7, 99);

    const [, path] = mockSendRequest.mock.calls[0];
    expect(path).toContain("installer_id=99");
    expect(path).toContain("fleet_id=7");
  });
});

describe("softwareAPI.addAppStoreApp", () => {
  beforeEach(() => {
    mockSendRequest.mockReset();
    mockSendRequest.mockResolvedValue({ software_title_id: 1 });
  });

  const vppFormData = {
    selfService: false,
    automaticInstall: false,
    targetType: "All hosts",
    customTarget: "labelsIncludeAny",
    labelTargets: {},
    categories: [],
    selectedApp: createMockVppApp({ app_store_id: "123", platform: "ios" }),
  };

  it("sends vpp_token_id for Apple apps", async () => {
    await softwareAPI.addAppStoreApp(7, { ...vppFormData, vppTokenId: 2 });

    const [method, , body] = mockSendRequest.mock.calls[0];
    expect(method).toBe("POST");
    expect(body).toMatchObject({
      app_store_id: "123",
      fleet_id: 7,
      vpp_token_id: 2,
    });
  });

  it("does not send vpp_token_id for Android apps", async () => {
    await softwareAPI.addAppStoreApp(7, {
      applicationID: "com.example.app",
      platform: "android",
      selfService: false,
      automaticInstall: false,
      targetType: "All hosts",
      customTarget: "labelsIncludeAny",
      labelTargets: {},
      categories: [],
    });

    const [, , body] = mockSendRequest.mock.calls[0];
    expect(body).not.toHaveProperty("vpp_token_id");
  });
});
