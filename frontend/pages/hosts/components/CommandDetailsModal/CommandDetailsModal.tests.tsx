import React from "react";
import { render, screen } from "@testing-library/react";

import { ICommandResult } from "interfaces/command";

import {
  getIconName,
  getVerbForCommandStatus,
  ModalContent,
  UnlockUserAccountCommandStatus,
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

describe("UnlockUserAccountCommandStatus", () => {
  const result = (status: string): ICommandResult => ({
    host_uuid: "host-uuid",
    command_uuid: "command-uuid",
    status,
    updated_at: "",
    request_type: "UnlockUserAccount",
    hostname: "Anna's Mac",
    payload: "",
    result: "",
    name: null,
  });

  it.each([
    ["Pending", /request to unlock.*is pending/i],
    ["NotNow", /request to unlock.*is deferred/i],
    ["Acknowledged", /unlocked the.*user account/i],
    ["Error", /failed to unlock the.*user account/i],
  ])("renders dedicated copy for %s", (status, expectedCopy) => {
    render(
      <UnlockUserAccountCommandStatus
        result={result(status)}
        username="anna"
        actorFullName="Jay Moore"
      />
    );

    expect(screen.getByText(expectedCopy)).toBeInTheDocument();
    expect(screen.queryByText(/custom MDM command/i)).not.toBeInTheDocument();
  });

  it("uses activity metadata when the command result was deleted", () => {
    render(
      <UnlockUserAccountCommandStatus
        result={{ ...result("Deleted"), hostname: "", request_type: "" }}
        username="anna"
        actorFullName="Jay Moore"
        hostDisplayName="Anna's Mac"
      />
    );

    expect(
      screen.getByText(/sent a request to unlock the.*user account/i)
    ).toBeInTheDocument();
    expect(screen.getByText("Anna's Mac")).toBeInTheDocument();
    expect(
      screen.getByText("This command has been deleted.")
    ).toBeInTheDocument();
    expect(screen.queryByText(/custom MDM command/i)).not.toBeInTheDocument();
  });
});
