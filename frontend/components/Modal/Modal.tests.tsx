import React from "react";
import { noop } from "lodash";
import { render, screen, fireEvent } from "@testing-library/react";

import Modal from "./Modal";

describe("Modal", () => {
  it("renders title", () => {
    render(
      <Modal title="Foobar" onExit={noop}>
        <div>test</div>
      </Modal>
    );

    expect(screen.getByText("Foobar")).toBeVisible();
  });

  it("calls onExit when clicking the background overlay", () => {
    const onExit = jest.fn();
    const { container } = render(
      <Modal title="Test" onExit={onExit}>
        <div>content</div>
      </Modal>
    );

    const background = container.querySelector(".modal__background");
    fireEvent.click(background!);
    expect(onExit).toHaveBeenCalledTimes(1);
  });

  it("does not call onExit when clicking inside the modal container", () => {
    const onExit = jest.fn();
    render(
      <Modal title="Test" onExit={onExit}>
        <div>content</div>
      </Modal>
    );

    fireEvent.click(screen.getByText("content"));
    expect(onExit).not.toHaveBeenCalled();
  });

  it("does not call onExit when clicking the background if disableClosingModal is true", () => {
    const onExit = jest.fn();
    const { container } = render(
      <Modal title="Test" onExit={onExit} disableClosingModal>
        <div>content</div>
      </Modal>
    );

    const background = container.querySelector(".modal__background");
    fireEvent.click(background!);
    expect(onExit).not.toHaveBeenCalled();
  });
});
