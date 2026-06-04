import React from "react";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import mockServer from "test/mock-server";
import {
  emptySelfServiceCategoriesHandler,
  listSelfServiceCategoriesErrorHandler,
  listSelfServiceCategoriesHandler,
} from "test/handlers/self-service-categories-handlers";

import SoftwareOptionsSelector from "./SoftwareOptionsSelector";

const defaultProps = {
  formData: {
    selfService: false,
    automaticInstall: false,
    targetType: "",
    customTarget: "",
    labelTargets: {},
    selectedApp: null,
    categories: [],
  },
  onToggleAutomaticInstall: jest.fn(),
  onToggleSelfService: jest.fn(),
  onSelectCategory: jest.fn(),
  onClickPreviewEndUserExperience: jest.fn(),
};

const getSwitchByLabelText = (text: string) => {
  const label = screen.getByText(text);
  const wrapper = label.closest(".fleet-slider__wrapper");
  if (!wrapper) throw new Error(`Wrapper not found for "${text}"`);
  const btn = wrapper.querySelector('button[role="switch"]');
  if (!btn) throw new Error(`Switch button not found for "${text}"`);
  return btn as HTMLButtonElement;
};

describe("SoftwareOptionsSelector", () => {
  const renderComponent = (props = {}) => {
    return createCustomRenderer({ context: {}, withBackendMock: true })(
      <SoftwareOptionsSelector {...defaultProps} {...props} />
    );
  };

  it("calls onToggleSelfService when the self-service slider is toggled", () => {
    const onToggleSelfService = jest.fn();
    renderComponent({ onToggleSelfService });

    const selfServiceSwitch = getSwitchByLabelText("Self-service");
    fireEvent.click(selfServiceSwitch);

    expect(onToggleSelfService).toHaveBeenCalledTimes(1);
    // Slider calls onChange with no args
    expect(onToggleSelfService).toHaveBeenCalledWith();
  });

  it("enables self-service sliders for iOS", () => {
    renderComponent({ platform: "ios" });

    const selfServiceSwitch = getSwitchByLabelText("Self-service");
    expect(selfServiceSwitch.disabled).toBe(false);
  });

  it("enables self-service  for iPadOS", () => {
    renderComponent({ platform: "ipados" });

    const selfServiceSwitch = getSwitchByLabelText("Self-service");
    expect(selfServiceSwitch.disabled).toBe(false);
  });

  it("disables self-service when disableOptions is true", () => {
    renderComponent({ disableOptions: true });

    const selfServiceSwitch = getSwitchByLabelText("Self-service");

    expect(selfServiceSwitch.disabled).toBe(true);
  });

  describe("dynamic categories (teamId provided)", () => {
    const selfServiceEditingProps = {
      ...defaultProps,
      formData: { ...defaultProps.formData, selfService: true },
      isEditingSoftware: true,
      teamId: 1,
    };

    it("renders categories returned by the API as checkboxes", async () => {
      mockServer.use(
        listSelfServiceCategoriesHandler([
          { id: 1, name: "🌎 Browsers" },
          { id: 2, name: "🔐 Security" },
        ])
      );

      renderComponent(selfServiceEditingProps);

      expect(await screen.findByText("🌎 Browsers")).toBeInTheDocument();
      expect(screen.getByText("🔐 Security")).toBeInTheDocument();
    });

    it("treats teamId 0 (no team) as dynamic, fetching categories from the API", async () => {
      // A name absent from the hardcoded fallback proves teamId 0 queried the API.
      mockServer.use(
        listSelfServiceCategoriesHandler([
          { id: 9, name: "🛟 No-team custom category" },
        ])
      );

      renderComponent({ ...selfServiceEditingProps, teamId: 0 });

      expect(
        await screen.findByText("🛟 No-team custom category")
      ).toBeInTheDocument();
    });

    it("shows the empty state with an Add category link when no categories exist", async () => {
      mockServer.use(emptySelfServiceCategoriesHandler);

      renderComponent(selfServiceEditingProps);

      const link = await screen.findByRole("link", { name: /add category/i });
      expect(link).toHaveAttribute("href", "/software/library/categories");
      expect(screen.getByText("to assign software to it.")).toBeInTheDocument();
    });

    it("does not render the empty state while categories are loading", () => {
      mockServer.use(emptySelfServiceCategoriesHandler);

      renderComponent(selfServiceEditingProps);

      expect(
        screen.queryByText("to assign software to it.")
      ).not.toBeInTheDocument();
    });

    it("renders a data error and not the empty state when the API errors", async () => {
      mockServer.use(listSelfServiceCategoriesErrorHandler);

      renderComponent(selfServiceEditingProps);

      expect(
        await screen.findByText(/something's gone wrong/i)
      ).toBeInTheDocument();
      expect(
        screen.queryByText("to assign software to it.")
      ).not.toBeInTheDocument();
    });

    it("fires onSelectCategory with the API category name when a checkbox is toggled", async () => {
      mockServer.use(
        listSelfServiceCategoriesHandler([{ id: 1, name: "🌎 Browsers" }])
      );
      const onSelectCategory = jest.fn();

      renderComponent({ ...selfServiceEditingProps, onSelectCategory });

      const checkbox = await screen.findByRole("checkbox", {
        name: "🌎 Browsers",
      });
      fireEvent.click(checkbox);

      expect(onSelectCategory).toHaveBeenCalledTimes(1);
      expect(onSelectCategory).toHaveBeenCalledWith({
        name: "🌎 Browsers",
        value: true,
      });
    });

    it("renders pre-selected categories as checked", async () => {
      mockServer.use(
        listSelfServiceCategoriesHandler([
          { id: 1, name: "🌎 Browsers" },
          { id: 2, name: "🔐 Security" },
        ])
      );

      renderComponent({
        ...selfServiceEditingProps,
        formData: {
          ...selfServiceEditingProps.formData,
          categories: ["🔐 Security"],
        },
      });

      const security = await screen.findByRole("checkbox", {
        name: "🔐 Security",
      });
      const browsers = await screen.findByRole("checkbox", {
        name: "🌎 Browsers",
      });

      expect(security).toHaveAttribute("aria-checked", "true");
      expect(browsers).toHaveAttribute("aria-checked", "false");
    });
  });

  describe("category list visibility (canSelectSoftwareCategories)", () => {
    it("does not render the categories list when self-service is off, even with a teamId", () => {
      mockServer.use(
        listSelfServiceCategoriesHandler([{ id: 1, name: "🌎 Browsers" }])
      );

      renderComponent({
        ...defaultProps,
        formData: { ...defaultProps.formData, selfService: false },
        isEditingSoftware: true,
        teamId: 1,
      });

      expect(screen.queryByText("Categories")).not.toBeInTheDocument();
      expect(screen.queryByText("🌎 Browsers")).not.toBeInTheDocument();
    });

    it("does not render the categories list when not in edit mode, even with self-service on and a teamId", () => {
      mockServer.use(
        listSelfServiceCategoriesHandler([{ id: 1, name: "🌎 Browsers" }])
      );

      renderComponent({
        ...defaultProps,
        formData: { ...defaultProps.formData, selfService: true },
        isEditingSoftware: false,
        teamId: 1,
      });

      expect(screen.queryByText("Categories")).not.toBeInTheDocument();
      expect(screen.queryByText("🌎 Browsers")).not.toBeInTheDocument();
    });
  });

  describe("static categories (no teamId)", () => {
    it("renders the hardcoded category list when teamId is not provided", async () => {
      const props = {
        ...defaultProps,
        formData: { ...defaultProps.formData, selfService: true },
        isEditingSoftware: true,
      };

      renderComponent(props);

      // Wait a tick so any pending dynamic fetch would have surfaced
      await waitFor(() =>
        expect(screen.getByText("🌎 Browsers")).toBeInTheDocument()
      );
      expect(screen.getByText("👬 Communication")).toBeInTheDocument();
    });
  });
});
