import React from "react";

import {
  createMockSoftwareTitle,
  createMockSoftwarePackage,
  createMockAppStoreApp,
  createMockAppStoreAppAndroid,
} from "__mocks__/softwareMock";

import { render as defaultRender, screen } from "@testing-library/react";
import { UserEvent } from "@testing-library/user-event";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import EditAutoUpdateConfigModal from "./EditAutoUpdateConfigModal";

const router = createMockRouter();

jest.mock("../../components/icons/SoftwareIcon", () => {
  return {
    __esModule: true,
    default: () => {
      return <div />;
    },
  };
});

describe("Edit Auto Update Config Modal", () => {
  describe("Auto updates options", () => {
    it("Does not show maintenance window options when 'Enable auto updates' is unchecked", async () => {});
    it("Shows maintenance window options when 'Enable auto updates' is checked", async () => {});
    describe("Maintenance window validation", () => {
      it("Requires start time to be HH:MM format", async () => {});
      it("Requires end time to be HH:MM format", async () => {});
      it("Requires both start and end times to be set", async () => {});
      it("Requires window to be at least one hour", async () => {});
    });
  });
  describe("Target options", () => {
    it("Shows 'All hosts' if no labels are configured for the title", async () => {});
    it("Shows label options if labels are configured for the title", async () => {});
  });
  describe("Submitting the form", () => {
    it("Sends the correct payload when 'Enable auto updates' is unchecked", async () => {});
    it("Sends the correct payload when 'Enable auto updates' is checked and a valid window is configured", async () => {});
    it("Sends the correct payload when 'All hosts' is selected as the target", async () => {});
    it("Sends the correct payload when specific labels are selected as the target", async () => {});
  });
});
