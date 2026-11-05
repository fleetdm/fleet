import { act, fireEvent, render, screen } from "@testing-library/react";
import React from "react";

import ColumnPreferencesModal from "./ColumnPreferencesModal";

describe("ColumnPreferencesModal keyboard focus", () => {
  it("focuses a dialog control when no column can move", () => {
    render(
      <ColumnPreferencesModal
        columns={["Host"]}
        onSave={jest.fn()}
        onExit={jest.fn()}
      />
    );
    const cancel = screen.getByRole("button", { name: "Cancel" });
    expect(cancel.closest(".modal__modal_container")).toContainElement(
      document.activeElement as HTMLElement
    );
    expect(
      screen.getByRole("button", { name: "Move up: Host" })
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: "Move down: Host" })
    ).toBeDisabled();
  });

  it("contains Tab in both directions while saving and when the dialog stays open afterward", async () => {
    let finishSave: () => void = () => undefined;
    const onSave = jest.fn(
      () =>
        new Promise<void>((resolve) => {
          finishSave = resolve;
        })
    );
    render(
      <ColumnPreferencesModal
        columns={["Host"]}
        onSave={onSave}
        onExit={jest.fn()}
      />
    );
    const save = screen.getByRole("button", { name: "Save" });
    save.focus();
    fireEvent.click(save);
    const dialog = save.closest(".modal__modal_container") as HTMLElement;
    expect(save).toBeDisabled();
    [false, true].forEach((shiftKey) => {
      expect(
        fireEvent.keyDown(document.activeElement as HTMLElement, {
          key: "Tab",
          shiftKey,
        })
      ).toBe(false);
      expect(dialog).toHaveFocus();
    });
    await act(async () => finishSave());
    fireEvent.keyDown(dialog, { key: "Tab", shiftKey: true });
    expect(screen.getByRole("button", { name: "Cancel" })).toHaveFocus();
  });
});
