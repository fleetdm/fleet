import React from "react";
import { render, screen } from "@testing-library/react";

import {
  createMockCommand,
  createMockGetCommandsResponse,
} from "__mocks__/commandMock";

import CommandFeed from "./CommandFeed";

describe("CommandFeed", () => {
  const mockOnShowDetails = jest.fn();
  const mockOnNextPage = jest.fn();
  const mockOnPreviousPage = jest.fn();

  const defaultProps = {
    emptyDescription: "Completed MDM commands will appear here.",
    onShowDetails: mockOnShowDetails,
    onNextPage: mockOnNextPage,
    onPreviousPage: mockOnPreviousPage,
  };

  it("renders empty state when results is null", () => {
    const commands = createMockGetCommandsResponse({
      results: null,
    });

    render(<CommandFeed commands={commands} {...defaultProps} />);

    expect(screen.getByText("No MDM commands")).toBeInTheDocument();
    expect(
      screen.getByText("Completed MDM commands will appear here.")
    ).toBeInTheDocument();
  });

  it("renders empty state when results is an empty array", () => {
    const commands = createMockGetCommandsResponse({
      results: [],
    });

    render(<CommandFeed commands={commands} {...defaultProps} />);

    expect(screen.getByText("No MDM commands")).toBeInTheDocument();
    expect(
      screen.getByText("Completed MDM commands will appear here.")
    ).toBeInTheDocument();
  });

  it("does not render pagination when has_next_results and has_previous_results are both false", () => {
    const commands = createMockGetCommandsResponse({
      meta: {
        has_next_results: false,
        has_previous_results: false,
      },
    });

    render(<CommandFeed commands={commands} {...defaultProps} />);

    const prevButton = screen.queryByText("Previous");
    const nextButton = screen.queryByText("Next");

    expect(prevButton).not.toBeInTheDocument();
    expect(nextButton).not.toBeInTheDocument();
  });

  it("renders pagination when there are more pagination items", () => {
    const commands = createMockGetCommandsResponse({
      meta: {
        has_next_results: true,
        has_previous_results: true,
      },
    });

    render(<CommandFeed commands={commands} {...defaultProps} />);

    expect(screen.getByText("Previous")).toBeInTheDocument();
    expect(screen.getByText("Next")).toBeInTheDocument();
  });

  it("renders a cancel button only on cancelable pending commands when onCancelCommand is provided", () => {
    const commands = createMockGetCommandsResponse({
      results: [
        createMockCommand({
          command_uuid: "lock-uuid",
          request_type: "DeviceLock",
        }),
        createMockCommand({
          command_uuid: "profile-uuid",
          request_type: "InstallProfile",
        }),
      ],
    });

    render(
      <CommandFeed
        commands={commands}
        {...defaultProps}
        onCancelCommand={jest.fn()}
      />
    );

    expect(
      screen.getAllByRole("button", { name: "cancel action" })
    ).toHaveLength(1);
  });

  it("does not render cancel buttons on already-run commands even when onCancelCommand is provided", () => {
    const commands = createMockGetCommandsResponse({
      results: [
        createMockCommand({
          request_type: "DeviceLock",
          command_status: "ran",
          status: "Acknowledged",
        }),
      ],
    });

    render(
      <CommandFeed
        commands={commands}
        {...defaultProps}
        onCancelCommand={jest.fn()}
      />
    );

    expect(
      screen.queryByRole("button", { name: "cancel action" })
    ).not.toBeInTheDocument();
  });

  it("does not render cancel buttons without onCancelCommand", () => {
    const commands = createMockGetCommandsResponse({
      results: [createMockCommand({ request_type: "DeviceLock" })],
    });

    render(<CommandFeed commands={commands} {...defaultProps} />);

    expect(
      screen.queryByRole("button", { name: "cancel action" })
    ).not.toBeInTheDocument();
  });
});
