import React from "react";
import { noop } from "lodash";
import { render, screen, fireEvent, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import Modal from "./Modal";

const clickBackground = async (background: Element | null) => {
  if (!background) throw new Error("Background element not found");
  const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime });
  await user.pointer([
    { keys: "[MouseLeft>]", target: background },
    { keys: "[/MouseLeft]", target: background },
  ]);
};

describe("Modal", () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it("renders title", () => {
    render(
      <Modal title="Foobar" onExit={noop}>
        <div>test</div>
      </Modal>
    );

    expect(screen.getByText("Foobar")).toBeVisible();
  });

  it("calls onExit when clicking the background overlay", async () => {
    const onExit = jest.fn();
    const { container } = render(
      <Modal title="Test" onExit={onExit}>
        <div>content</div>
      </Modal>
    );

    const background = container.querySelector(".modal__background");
    await clickBackground(background);
    act(() => jest.runAllTimers());
    expect(onExit).toHaveBeenCalledTimes(1);
  });

  it("does not call onExit when clicking inside the modal container", () => {
    const onExit = jest.fn();
    const { container } = render(
      <Modal title="Test" onExit={onExit}>
        <div>content</div>
      </Modal>
    );

    // Simulate drag that starts inside the container and releases on the background.
    // The container stops mouseDown propagation so isDownOnBackgroundRef stays false,
    // meaning the background's mouseUp handler must not fire onExit.
    fireEvent.mouseDown(screen.getByText("content"));
    const background = container.querySelector(".modal__background");
    if (!background) throw new Error("Background element not found");
    fireEvent.mouseUp(background);
    expect(onExit).not.toHaveBeenCalled();
  });

  it("does not call onExit when clicking the background if a form inside has been interacted with", async () => {
    const onExit = jest.fn();
    const { container } = render(
      <Modal title="Test" onExit={onExit}>
        <form>
          <input type="text" />
        </form>
      </Modal>
    );

    const input = screen.getByRole("textbox");
    fireEvent.input(input, { target: { value: "hello" } });

    const background = container.querySelector(".modal__background");
    await clickBackground(background);
    act(() => jest.runAllTimers());
    expect(onExit).not.toHaveBeenCalled();
  });

  it("does not call onExit when clicking the background if a text input outside a form has been interacted with", async () => {
    const onExit = jest.fn();
    const { container } = render(
      <Modal title="Test" onExit={onExit}>
        <div>
          <input type="text" />
        </div>
      </Modal>
    );

    const input = screen.getByRole("textbox");
    fireEvent.input(input, { target: { value: "hello" } });

    const background = container.querySelector(".modal__background");
    await clickBackground(background);
    act(() => jest.runAllTimers());
    expect(onExit).not.toHaveBeenCalled();
  });

  it("does not call onExit when clicking the background if a checkbox has been checked", async () => {
    const onExit = jest.fn();
    const { container } = render(
      <Modal title="Test" onExit={onExit}>
        <div>
          <input type="checkbox" />
        </div>
      </Modal>
    );

    const checkbox = screen.getByRole("checkbox");
    fireEvent.click(checkbox);

    const background = container.querySelector(".modal__background");
    await clickBackground(background);
    act(() => jest.runAllTimers());
    expect(onExit).not.toHaveBeenCalled();
  });

  it("does not call onExit when clicking the background if a toggle has been clicked", async () => {
    const onExit = jest.fn();
    const { container } = render(
      <Modal title="Test" onExit={onExit}>
        <div>
          <button type="button" role="switch" aria-checked={false}>
            Toggle
          </button>
        </div>
      </Modal>
    );

    const toggle = screen.getByRole("switch");
    fireEvent.click(toggle);

    const background = container.querySelector(".modal__background");
    await clickBackground(background);
    act(() => jest.runAllTimers());
    expect(onExit).not.toHaveBeenCalled();
  });

  it("calls onExit when clicking the background if a form inside has not been interacted with", async () => {
    const onExit = jest.fn();
    const { container } = render(
      <Modal title="Test" onExit={onExit}>
        <form>
          <input type="text" />
        </form>
      </Modal>
    );

    const background = container.querySelector(".modal__background");
    await clickBackground(background);
    act(() => jest.runAllTimers());
    expect(onExit).toHaveBeenCalledTimes(1);
  });

  it("does not call onExit when a backdrop mousedown is interrupted by the window losing focus", () => {
    const onExit = jest.fn();
    const { container } = render(
      <Modal title="Test" onExit={onExit}>
        <div>content</div>
      </Modal>
    );

    const background = container.querySelector(".modal__background");
    if (!background) throw new Error("Background element not found");

    // User presses down on the backdrop then the window loses focus (tab switch /
    // app switch) before releasing — mouseup never fires on the page.
    fireEvent.mouseDown(background);
    fireEvent.blur(window);
    // On return, the stale mousedown state must not close the modal.
    fireEvent.mouseUp(background);
    act(() => jest.runAllTimers());

    expect(onExit).not.toHaveBeenCalled();
  });

  // Modal groups hide and re-show modals via isHidden without unmounting them.
  // An animated close (X, Escape, backdrop) must fully reset, or the next show
  // renders stuck at the end of the fade-out: invisible, click-blocking, and
  // with every further close attempt ignored. See #51497.
  it("can be shown and closed again after an animated close when reused via isHidden", async () => {
    const onExit = jest.fn();
    const { container, rerender } = render(
      <Modal title="Test" onExit={onExit}>
        <div>content</div>
      </Modal>
    );

    const closeButton = container.querySelector(".modal__ex button");
    if (!closeButton) throw new Error("X button not found");

    fireEvent.click(closeButton);
    act(() => jest.runAllTimers());
    expect(onExit).toHaveBeenCalledTimes(1);

    // parent hides it, then shows it again — no unmount in between
    rerender(
      <Modal title="Test" onExit={onExit} isHidden>
        <div>content</div>
      </Modal>
    );
    rerender(
      <Modal title="Test" onExit={onExit}>
        <div>content</div>
      </Modal>
    );

    const background = container.querySelector(".modal__background");
    expect(background).not.toHaveClass("modal__closing");

    // and the X must work again, not be swallowed by stale closing state
    fireEvent.click(closeButton);
    act(() => jest.runAllTimers());
    expect(onExit).toHaveBeenCalledTimes(2);
  });

  // A modal whose close swaps to another modal in a group closes instantly,
  // like a footer Close button — playing the exit animation first shows both
  // modals briefly. See #51497.
  it("closes immediately and repeatedly with disableExitAnimation", () => {
    const onExit = jest.fn();
    const { container } = render(
      <Modal title="Test" onExit={onExit} disableExitAnimation>
        <div>content</div>
      </Modal>
    );

    const closeButton = container.querySelector(".modal__ex button");
    if (!closeButton) throw new Error("X button not found");

    fireEvent.click(closeButton);
    expect(onExit).toHaveBeenCalledTimes(1);

    const background = container.querySelector(".modal__background");
    expect(background).not.toHaveClass("modal__closing");

    // no stale closing state — a second close works without any timers
    fireEvent.click(closeButton);
    expect(onExit).toHaveBeenCalledTimes(2);
  });

  it("does not call onExit when clicking the background if disableClosingModal is true", async () => {
    const onExit = jest.fn();
    const { container } = render(
      <Modal title="Test" onExit={onExit} disableClosingModal>
        <div>content</div>
      </Modal>
    );

    const background = container.querySelector(".modal__background");
    await clickBackground(background);
    act(() => jest.runAllTimers());
    expect(onExit).not.toHaveBeenCalled();
  });
});
