import sendRequest from "services";

import labelsAPI, { listNamesFromSelectedLabels } from "./labels";
import createMockHost from "../../__mocks__/hostMock";

jest.mock("services", () => ({
  __esModule: true,
  default: jest.fn(),
}));

const mockSendRequest = sendRequest as jest.MockedFunction<typeof sendRequest>;

describe("labelsAPI.update", () => {
  beforeEach(() => {
    mockSendRequest.mockReset();
    mockSendRequest.mockResolvedValue({});
  });

  const manualFormData = {
    name: "Remediation exclusion",
    description: "Hosts temporarily excluded",
    targetedHosts: [createMockHost({ id: 7 }), createMockHost({ id: 9 })],
  };

  it("sends the definition alongside membership by default", async () => {
    await labelsAPI.update(1, manualFormData);

    expect(mockSendRequest).toHaveBeenCalledWith("PATCH", expect.any(String), {
      name: "Remediation exclusion",
      description: "Hosts temporarily excluded",
      host_ids: [7, 9],
    });
  });

  it("sends only membership when the definition is managed in git", async () => {
    await labelsAPI.update(1, manualFormData, { membershipOnly: true });

    expect(mockSendRequest).toHaveBeenCalledWith("PATCH", expect.any(String), {
      host_ids: [7, 9],
    });
  });

  it("sends an empty host list when the last host is removed", async () => {
    await labelsAPI.update(
      1,
      { ...manualFormData, targetedHosts: [] },
      { membershipOnly: true }
    );

    expect(mockSendRequest).toHaveBeenCalledWith("PATCH", expect.any(String), {
      host_ids: [],
    });
  });
});

describe("listNamesFromSelectedLabels", () => {
  it("returns names of selected labels", () => {
    expect(
      listNamesFromSelectedLabels({ foo: true, bar: false, baz: true })
    ).toEqual(["foo", "baz"]);
  });

  it("returns empty array when nothing is selected", () => {
    expect(listNamesFromSelectedLabels({ foo: false, bar: false })).toEqual([]);
  });

  it("returns empty array for an empty dict", () => {
    expect(listNamesFromSelectedLabels({})).toEqual([]);
  });
});
