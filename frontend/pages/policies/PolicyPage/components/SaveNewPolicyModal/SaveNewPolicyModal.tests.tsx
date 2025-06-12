import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import createMockQuery from "__mocks__/queryMock";
import createMockUser from "__mocks__/userMock";
import createMockConfig from "__mocks__/configMock";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";

import { ILabelSummary } from "interfaces/label";
import PolicyProvider from "context/policy";
import SaveNewPolicyModal from "./SaveNewPolicyModal";

const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

const mockLabels: ILabelSummary[] = [
  {
    id: 1,
    name: "Fun",
    description: "Computers that like to have a good time",
    label_type: "regular",
  },
  {
    id: 2,
    name: "Fresh",
    description: "Laptops with dirty mouths",
    label_type: "regular",
  },
];

const labelSummariesHandler = http.get(baseUrl("/labels/summary"), () => {
  return HttpResponse.json({
    labels: mockLabels,
  });
});

describe("SaveNewPolicyModal", () => {
  const defaultProps = {
    baseClass: "",
    queryValue: "",
    onCreatePolicy: jest.fn(),
    setIsSaveNewPolicyModalOpen: jest.fn(),
    backendValidators: {},
    platformSelector: {
      setSelectedPlatforms: jest.fn(),
      getSelectedPlatforms: () => {
        return [];
      },
      isAnyPlatformSelected: true,
      render: () => <div />,
      disabled: false,
    },
    isUpdatingPolicy: false,
    isFetchingAutofillDescription: false,
    isFetchingAutofillResolution: false,
    onClickAutofillDescription: jest.fn(),
    onClickAutofillResolution: jest.fn(),
    labels: mockLabels,
  };

  it("should not show the target selector in the free tier", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          currentUser: createMockUser(),
          config: createMockConfig(),
          isPremiumTier: false,
        },
      },
    });

    render(<SaveNewPolicyModal {...defaultProps} />);

    // Wait for any queries (that should not be happening) to finish.
    await new Promise((resolve) => setTimeout(resolve, 500));

    // Check that the target selector is not present.
    expect(screen.queryByText("All hosts")).not.toBeInTheDocument();
  });

  describe("in premium tier", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          currentUser: createMockUser(),
          isGlobalObserver: false,
          isGlobalAdmin: true,
          isGlobalMaintainer: false,
          isOnGlobalTeam: true,
          isPremiumTier: true,
          isSandboxMode: false,
          config: createMockConfig(),
        },
      },
    });

    beforeEach(() => {
      mockServer.use(labelSummariesHandler);
    });

    it("should show the target selector in All hosts target mode when the policy has no labels", async () => {
      render(<SaveNewPolicyModal {...defaultProps} />);
      await waitFor(() => {
        expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
        expect(screen.getByLabelText("Custom")).toBeInTheDocument();
        expect(screen.getByLabelText("All hosts")).toBeChecked();
      });
    });

    it("should disable the save button in Custom target mode when no labels are selected, and enable it once labels are selected", async () => {
      render(<SaveNewPolicyModal {...defaultProps} />);
      let allHosts;
      let custom;
      await waitFor(() => {
        allHosts = screen.getByLabelText("All hosts");
        custom = screen.getByLabelText("Custom");
        expect(allHosts).toBeInTheDocument();
        expect(custom).toBeInTheDocument();
      });
      custom && (await userEvent.click(custom));
      const saveButton = screen.getByRole("button", { name: "Save" });
      expect(saveButton).toBeDisabled();

      const funButton = screen.getByLabelText("Fun");
      expect(funButton).not.toBeChecked();
      await userEvent.click(funButton);
      expect(saveButton).toBeEnabled();
    });

    it("should send labels when saving a new policy in Custom target mode (include any)", async () => {
      const onCreatePolicy = jest.fn();
      const props = { ...defaultProps, onCreatePolicy };
      render(
        <PolicyProvider>
          <SaveNewPolicyModal {...props} />
        </PolicyProvider>
      );
      await waitFor(() => {
        expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
      });

      // Set a name.
      const nameInput = screen.getByLabelText("Name");
      await userEvent.type(nameInput, "A Brand New Policy!");

      // Set a label.
      await userEvent.click(screen.getByLabelText("Custom"));
      await userEvent.click(screen.getByLabelText("Fun"));
      await userEvent.click(screen.getByRole("button", { name: "Save" }));

      expect(onCreatePolicy.mock.calls[0][0].labels_include_any).toEqual([
        "Fun",
      ]);
    });

    it("should send labels when saving a new policy in Custom target mode (exclude any)", async () => {
      const onCreatePolicy = jest.fn();
      const props = { ...defaultProps, onCreatePolicy };
      render(
        <PolicyProvider>
          <SaveNewPolicyModal {...props} />
        </PolicyProvider>
      );
      await waitFor(() => {
        expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
      });

      // Set a name.
      const nameInput = screen.getByLabelText("Name");
      await userEvent.type(nameInput, "A Brand New Policy!");

      // Set a label.
      await userEvent.click(screen.getByLabelText("Custom"));
      await userEvent.click(screen.getByLabelText("Fun"));

      // Click "Include any" to open the dropdown.
      const includeAnyOption = screen.getByRole("option", {
        name: "Include any",
      });
      await userEvent.click(includeAnyOption);

      // Click "Exclude any" to select it.
      let excludeAnyOption: unknown;
      await waitFor(() => {
        excludeAnyOption = screen.getByRole("option", { name: "Exclude any" });
      });
      await userEvent.click(excludeAnyOption as Element);

      await userEvent.click(screen.getByRole("button", { name: "Save" }));

      expect(onCreatePolicy.mock.calls[0][0].labels_exclude_any).toEqual([
        "Fun",
      ]);
    });

    it("should clear labels when saving a new policy in All hosts target mode", async () => {
      const onCreatePolicy = jest.fn();
      const props = { ...defaultProps, onCreatePolicy };
      render(
        <PolicyProvider>
          <SaveNewPolicyModal {...props} />
        </PolicyProvider>
      );
      await waitFor(() => {
        expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
      });

      // Set a name.
      await userEvent.type(
        screen.getByLabelText("Name"),
        "A Brand New Policy!"
      );

      await userEvent.click(screen.getByRole("button", { name: "Save" }));

      expect(onCreatePolicy.mock.calls[0][0].labels_include_any).toEqual([]);
    });
  });
});
