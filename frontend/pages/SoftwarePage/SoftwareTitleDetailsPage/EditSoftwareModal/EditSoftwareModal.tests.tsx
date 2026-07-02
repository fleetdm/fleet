import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import { createMockSoftwarePackage } from "__mocks__/softwareMock";

import EditSoftwareModal from "./EditSoftwareModal";

const BASE_PROPS: React.ComponentProps<typeof EditSoftwareModal> = {
  softwareId: 1,
  teamId: 1,
  softwareInstaller: createMockSoftwarePackage(),
  refetchSoftwareTitle: jest.fn(),
  onExit: jest.fn(),
  installerType: "package",
  openViewYamlModal: jest.fn(),
  name: "GlobalProtect",
  displayName: "GlobalProtect",
  source: "apps",
  iconUrl: null,
};

const renderModal = (
  overrides: Partial<React.ComponentProps<typeof EditSoftwareModal>> = {}
) => {
  const render = createCustomRenderer({ withBackendMock: true });
  return render(<EditSoftwareModal {...BASE_PROPS} {...overrides} />);
};

describe("EditSoftwareModal — multi-package title (#48400)", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders the 'Edit software' title by default (single-package legacy path)", () => {
    renderModal();
    expect(screen.getByText("Edit software")).toBeInTheDocument();
    expect(screen.queryByText("Edit package")).not.toBeInTheDocument();
  });

  it("renders the 'Edit package' title when canActivateMultiplePackages is true", () => {
    renderModal({ canActivateMultiplePackages: true });
    expect(screen.getByText("Edit package")).toBeInTheDocument();
    expect(screen.queryByText("Edit software")).not.toBeInTheDocument();
  });

  it("accepts an installerId prop (threaded into the API call on save)", () => {
    // Smoke test — the prop is optional and exists on the interface. We don't
    // submit here since that would require asserting against the mocked API
    // client; the page tests cover the submit path end-to-end.
    expect(() => renderModal({ installerId: 7 })).not.toThrow();
  });
});
