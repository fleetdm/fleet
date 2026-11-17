import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createMockHostGeolocation } from "__mocks__/hostMock";

import { HostMdmDeviceStatusUIState } from "../../helpers";

import LocationModal from "./LocationModal";

const makeIosDetails = (
  status: HostMdmDeviceStatusUIState,
  isIosOrIpadosHost = true
) => ({
  isIosOrIpadosHost,
  hostMdmDeviceStatus: status,
});

describe("LocationModal", () => {
  it("renders basic location info and Google Maps link when hostGeolocation is provided", () => {
    render(
      <LocationModal
        hostGeolocation={createMockHostGeolocation()}
        onExit={noop}
      />
    );

    // Location name (city, country)
    expect(screen.getByText("Minneapolis, US")).toBeVisible();

    // Google Maps link built from coordinates (lat,lng)
    const link = screen.getByRole("link", { name: /Open in Google Maps/i });
    expect(link).toBeVisible();
    expect(link).toHaveAttribute(
      "href",
      "https://www.google.com/maps?q=-93.2602,44.9844"
    );
  });

  it("renders LastUpdatedText when detailsUpdatedAt is 2 days ago", () => {
    const currentDate = new Date();
    currentDate.setDate(currentDate.getDate() - 2);
    const twoDaysAgo = currentDate.toISOString();

    render(
      <LocationModal
        hostGeolocation={createMockHostGeolocation()}
        detailsUpdatedAt={twoDaysAgo}
        onExit={noop}
      />
    );

    expect(screen.getByText(/Updated 2 days ago/i)).toBeInTheDocument();
  });

  it("shows iOS unlocked message when iOS host is unlocked", () => {
    render(
      <LocationModal
        iosOrIpadosDetails={makeIosDetails("unlocked")}
        onExit={noop}
      />
    );

    expect(
      screen.getByText(
        /To view location, Apple requires that iOS\/iPadOS hosts are locked \(put in Lost Mode\) first./i
      )
    ).toBeVisible();
  });

  it("shows iOS locking message when lock is pending", () => {
    render(
      <LocationModal
        iosOrIpadosDetails={makeIosDetails("locking")}
        onExit={noop}
      />
    );

    expect(
      screen.getByText(
        /To view location, Apple requires that iOS\/iPadOS hosts are locked \(put in Lost Mode\) first./i
      )
    ).toBeVisible();
    expect(
      screen.getByText(
        /Lock is pending. Host will lock the next time it checks in to Fleet./i
      )
    ).toBeVisible();
  });

  it("shows iOS locating message when location is pending", () => {
    render(
      <LocationModal
        iosOrIpadosDetails={makeIosDetails("locating")}
        onExit={noop}
      />
    );

    expect(
      screen.getByText(
        /Location is pending. Host will share location the next time it checks in to Fleet./i
      )
    ).toBeVisible();
  });

  it("shows refetch message when iOS host has no location and status is unrecognized/empty", () => {
    render(
      <LocationModal
        iosOrIpadosDetails={makeIosDetails("wiping")}
        // no hostGeolocation => hasLocation = false
        onExit={noop}
      />
    );

    expect(screen.getByText(/Location not available/i)).toBeVisible();
    expect(screen.getByText(/Close this modal/i)).toBeVisible();
    expect(screen.getByText("Refetch")).toBeVisible();
  });

  it("falls back to normal location view when iOS host has a location and status is unrecognized/empty", () => {
    render(
      <LocationModal
        hostGeolocation={createMockHostGeolocation()}
        iosOrIpadosDetails={makeIosDetails("wiping")}
        onExit={noop}
      />
    );

    // With a location present, the standard content renders
    expect(screen.getByText("Minneapolis, US")).toBeVisible();
  });

  it("tells the user where to lock and renders only a Close button when iOS host is unlocked", () => {
    render(
      <LocationModal
        iosOrIpadosDetails={makeIosDetails("unlocked")}
        onExit={noop}
      />
    );

    expect(screen.getByText(/To lock, go to/i)).toBeVisible();
    expect(screen.getByText("Actions > Lock")).toBeVisible();
    expect(screen.getByRole("button", { name: "Close" })).toBeVisible();
    expect(
      screen.queryByRole("button", { name: "Lock" })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Cancel" })
    ).not.toBeInTheDocument();
  });

  it("does not tell the user where to lock when lock is pending", () => {
    render(
      <LocationModal
        iosOrIpadosDetails={makeIosDetails("locking")}
        onExit={noop}
      />
    );

    expect(screen.queryByText(/To lock, go to/i)).not.toBeInTheDocument();
  });

  it("renders Close footer button otherwise", () => {
    render(
      <LocationModal
        hostGeolocation={createMockHostGeolocation()}
        // non‑iOS host
        iosOrIpadosDetails={makeIosDetails("unlocked", false)}
        onExit={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Close" })).toBeVisible();
  });
});
