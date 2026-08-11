import React from "react";
import { render, screen } from "@testing-library/react";

import {
  getIconName,
  getVerbForCommandStatus,
  ModalContent,
} from "./CommandDetailsModal";

describe("getIconName", () => {
  it("returns error for Apple Error status", () => {
    expect(getIconName("Error")).toEqual("error");
  });

  it("returns error for Apple CommandFormatError status", () => {
    expect(getIconName("CommandFormatError")).toEqual("error");
  });

  it("returns success for Apple Acknowledged status", () => {
    expect(getIconName("Acknowledged")).toEqual("success");
  });

  it("returns pending-outline for Apple Pending status", () => {
    expect(getIconName("Pending")).toEqual("pending-outline");
  });

  it("returns pending-outline for Apple NotNow status", () => {
    expect(getIconName("NotNow")).toEqual("pending-outline");
  });

  it("returns success for Windows 200 status", () => {
    expect(getIconName("200")).toEqual("success");
  });

  it("returns error for Windows 400 status", () => {
    expect(getIconName("400")).toEqual("error");
  });

  it("returns error for Windows 500 status", () => {
    expect(getIconName("500")).toEqual("error");
  });

  it("returns pending-outline for Windows 101 status", () => {
    expect(getIconName("101")).toEqual("pending-outline");
  });

  it("returns pending-outline for Windows 199 status (upper pending boundary)", () => {
    expect(getIconName("199")).toEqual("pending-outline");
  });

  it("returns success for Windows 399 status (upper success boundary)", () => {
    expect(getIconName("399")).toEqual("success");
  });

  it("returns warning for an unknown status", () => {
    expect(getIconName("unknown")).toEqual("warning");
  });
});

describe("getVerbForCommandStatus", () => {
  it("returns 'ran' for a successful status", () => {
    expect(getVerbForCommandStatus("Acknowledged")).toEqual("ran");
  });

  it("returns 'failed to run' for an error status", () => {
    expect(getVerbForCommandStatus("Error")).toEqual("failed to run");
  });

  it("returns 'sent' for a pending status", () => {
    expect(getVerbForCommandStatus("Pending")).toEqual("sent");
  });

  it("returns 'sent' for an unknown status", () => {
    expect(getVerbForCommandStatus("unknown")).toEqual("sent");
  });
});

describe("ModalContent", () => {
  it("renders normally, not as an error, when the API returns a 200 with no results (e.g. host re-enrolled since the command was sent)", () => {
    render(
      <ModalContent data={{ results: [] }} isLoading={false} error={null} />
    );

    expect(
      screen.getByText("This command has been deleted.")
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/something's gone wrong/i)
    ).not.toBeInTheDocument();
  });
});
