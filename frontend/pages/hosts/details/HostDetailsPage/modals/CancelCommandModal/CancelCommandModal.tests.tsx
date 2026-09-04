import React from "react";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createCustomRenderer } from "test/test-utils";

import { createMockCommand } from "__mocks__/commandMock";
import commandsAPI from "services/entities/command";

import CancelCommandModal from "./CancelCommandModal";

describe("CancelCommandModal", () => {
  const mockCommand = createMockCommand({
    command_uuid: "cmd-uuid-1",
    request_type: "DeviceLock",
  });

  const defaultProps = {
    hostId: 7,
    command: mockCommand,
    onSuccessCancel: jest.fn(),
    onExit: jest.fn(),
  };

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("renders the title, body copy, and command preview", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<CancelCommandModal {...defaultProps} />);

    expect(screen.getByText("Cancel upcoming command")).toBeInTheDocument();
    expect(
      screen.getByText(
        /If the activity is happening on the host it will still complete/i
      )
    ).toBeInTheDocument();
    expect(screen.getByText("DeviceLock")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Cancel command" })
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Back" })).toBeInTheDocument();
  });

  it("cancels the command and fires onSuccessCancel on success", async () => {
    const cancelSpy = jest
      .spyOn(commandsAPI, "cancelHostCommand")
      .mockResolvedValue(undefined);

    const render = createCustomRenderer({ withBackendMock: true });
    render(<CancelCommandModal {...defaultProps} />);

    await userEvent.click(
      screen.getByRole("button", { name: "Cancel command" })
    );

    await waitFor(() => {
      expect(cancelSpy).toHaveBeenCalledWith(7, "cmd-uuid-1");
    });
    expect(defaultProps.onSuccessCancel).toHaveBeenCalledWith(mockCommand);
    expect(defaultProps.onExit).toHaveBeenCalled();
  });

  it("does not fire onSuccessCancel when the API call fails", async () => {
    jest
      .spyOn(commandsAPI, "cancelHostCommand")
      .mockRejectedValue({ status: 400 });

    const render = createCustomRenderer({ withBackendMock: true });
    render(<CancelCommandModal {...defaultProps} />);

    await userEvent.click(
      screen.getByRole("button", { name: "Cancel command" })
    );

    await waitFor(() => {
      expect(defaultProps.onExit).toHaveBeenCalled();
    });
    expect(defaultProps.onSuccessCancel).not.toHaveBeenCalled();
  });

  it("closes without canceling when Back is clicked", async () => {
    const cancelSpy = jest.spyOn(commandsAPI, "cancelHostCommand");

    const render = createCustomRenderer({ withBackendMock: true });
    render(<CancelCommandModal {...defaultProps} />);

    await userEvent.click(screen.getByRole("button", { name: "Back" }));

    expect(defaultProps.onExit).toHaveBeenCalled();
    expect(cancelSpy).not.toHaveBeenCalled();
  });
});
