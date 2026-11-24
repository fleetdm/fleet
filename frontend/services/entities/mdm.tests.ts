import sendRequest from "services";
import endpoints from "utilities/endpoints";

import mdmAPI from "./mdm";

jest.mock("services", () => ({
  __esModule: true,
  default: jest.fn(),
}));

const mockSendRequest = jest.mocked(sendRequest);

// The platform picks the agreement on the shared EULA endpoints. macOS sends none,
// as it always has, so older servers keep working for it.
describe("mdmAPI EULA requests", () => {
  const macFile = new File(["%PDF-1.7"], "eula.pdf", {
    type: "application/pdf",
  });
  const mdFile = new File(["# Terms"], "terms.md", { type: "text/markdown" });

  beforeEach(() => {
    jest.restoreAllMocks();
    mockSendRequest.mockReset();
    mockSendRequest.mockResolvedValue({});
  });

  // jsdom clones a File put into a FormData, so the file is checked at append.
  const expectUpload = () => {
    expect(mockSendRequest).toHaveBeenCalledTimes(1);
    expect(mockSendRequest).toHaveBeenCalledWith(
      "POST",
      endpoints.MDM_EULA_UPLOAD,
      expect.any(FormData)
    );
  };

  it("uploadWindowsEULA sends the platform in the form, next to the file", async () => {
    const append = jest.spyOn(FormData.prototype, "append");

    await mdmAPI.uploadWindowsEULA(mdFile);

    expectUpload();
    expect(append).toHaveBeenCalledWith("platform", "windows");
    expect(append).toHaveBeenCalledWith("eula", mdFile);
  });

  it("uploadEULA sends no platform", async () => {
    const append = jest.spyOn(FormData.prototype, "append");

    await mdmAPI.uploadEULA(macFile);

    expectUpload();
    expect(append).toHaveBeenCalledTimes(1);
    expect(append).toHaveBeenCalledWith("eula", macFile);
  });

  it("getWindowsEULAMetadata sends the platform in the query", async () => {
    await mdmAPI.getWindowsEULAMetadata();

    expect(mockSendRequest).toHaveBeenCalledTimes(1);
    expect(mockSendRequest).toHaveBeenCalledWith(
      "GET",
      `${endpoints.MDM_EULA_METADATA}?platform=windows`
    );
  });

  it("getEULAMetadata sends no platform", async () => {
    await mdmAPI.getEULAMetadata();

    expect(mockSendRequest).toHaveBeenCalledTimes(1);
    expect(mockSendRequest).toHaveBeenCalledWith(
      "GET",
      endpoints.MDM_EULA_METADATA
    );
  });
});
