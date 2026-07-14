import sendRequest from "services";
import softwareAPI from "./software";

jest.mock("services", () => ({
  __esModule: true,
  default: jest.fn(),
  sendRequestWithHeaders: jest.fn(),
  sendRequestWithProgressAndHeaders: jest.fn(),
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
