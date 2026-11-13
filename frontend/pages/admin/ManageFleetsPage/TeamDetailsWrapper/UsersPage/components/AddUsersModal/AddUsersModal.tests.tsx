import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import createMockTeam from "__mocks__/teamMock";

import AddUsersModal from "./AddUsersModal";

describe("AddUsersModal", () => {
  it("closes on Escape while the user search input is focused", async () => {
    const onCancel = jest.fn();
    render(
      <AddUsersModal
        team={createMockTeam()}
        disabledUsers={[]}
        onCancel={onCancel}
        onSubmit={noop}
        onCreateNewTeamUser={noop}
      />
    );

    const input = screen.getByRole("combobox");
    expect(input).toHaveFocus();

    // react-select v1 switches on keyCode, which userEvent does not set.
    fireEvent.keyDown(input, { key: "Escape", keyCode: 27 });

    await waitFor(() => expect(onCancel).toHaveBeenCalled());
  });
});
