import { render, screen } from "@testing-library/react";
import React from "react";

import { IActivityDetails } from "interfaces/activity";

import AppStoreDetailsModal from "./AppStoreDetailsModal";

const baseDetails: IActivityDetails = {
  software_title: "Slack",
  app_store_id: 618783545,
  self_service: false,
};

const renderModal = (detailsOverrides: Partial<IActivityDetails> = {}) =>
  render(
    <AppStoreDetailsModal
      details={{ ...baseDetails, ...detailsOverrides }}
      onCancel={jest.fn()}
    />
  );

describe("AppStoreDetailsModal", () => {
  it("renders the Version row when platform is mobile and version_name is set", () => {
    renderModal({ platform: "ios", version_name: "Production" });
    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("Production")).toBeInTheDocument();
  });

  it("omits the Version row on macOS even when version_name is set", () => {
    // #53641 scopes the versioned-app UI to iOS/iPadOS/Android; macOS activities
    // may carry version_name but we don't surface it here.
    renderModal({ platform: "darwin", version_name: "Production" });
    expect(screen.queryByText("Version")).not.toBeInTheDocument();
  });

  it("omits the Version row when version_name is empty", () => {
    renderModal({ platform: "ios", version_name: "" });
    expect(screen.queryByText("Version")).not.toBeInTheDocument();
  });

  it("labels the store id as 'App store ID' for iOS/iPadOS", () => {
    renderModal({ platform: "ipados" });
    expect(screen.getByText("App store ID")).toBeInTheDocument();
    expect(screen.queryByText("Google Play ID")).not.toBeInTheDocument();
  });

  it("labels the store id as 'Google Play ID' for Android", () => {
    renderModal({ platform: "android" });
    expect(screen.getByText("Google Play ID")).toBeInTheDocument();
    expect(screen.queryByText("App store ID")).not.toBeInTheDocument();
  });

  it("renders the Configuration editor for iOS when configuration is set", () => {
    // Editor's virtual rendering hides the inner text from DOM queries, so we
    // only assert the labelled Editor shows up (and skip-renders are covered
    // by the "unset" and "macOS" cases).
    renderModal({
      platform: "ios",
      version_name: "P",
      configuration: "<dict><key>Server</key><string>https://x</string></dict>",
    });
    expect(screen.getByText("Configuration")).toBeInTheDocument();
  });

  it("renders the Configuration editor for Android when configuration is a parsed object", () => {
    // Android configuration arrives as a parsed JSON object (axios); the
    // component is responsible for stringifying it before handing to Editor.
    renderModal({
      platform: "android",
      version_name: "P",
      configuration: { HomepageLocation: "https://fleetdm.com" },
    });
    expect(screen.getByText("Configuration")).toBeInTheDocument();
  });

  it("omits the Configuration editor when configuration is unset", () => {
    renderModal({ platform: "ios", version_name: "P" });
    expect(screen.queryByText("Configuration")).not.toBeInTheDocument();
  });

  it("omits the Configuration editor on macOS even if configuration is set", () => {
    renderModal({ platform: "darwin", configuration: "<dict/>" });
    expect(screen.queryByText("Configuration")).not.toBeInTheDocument();
  });
});
