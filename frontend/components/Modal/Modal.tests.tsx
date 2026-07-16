import React, { useState } from "react";
import { noop } from "lodash";
import {
  render,
  screen,
  fireEvent,
  act,
  waitFor,
} from "@testing-library/react";
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

  describe("focus management", () => {
    beforeEach(() => jest.useRealTimers());

    it("focuses the modal container on open, not any interactive child", async () => {
      const { container } = render(
        <Modal title="Confirm" onExit={noop}>
          <>
            <button type="button">Submit</button>
            <button type="button">Cancel</button>
          </>
        </Modal>
      );

      const modalContainer = container.querySelector(".modal__modal_container");
      await waitFor(() => {
        expect(modalContainer).toHaveFocus();
      });
      expect(screen.getByRole("button", { name: "Submit" })).not.toHaveFocus();
    });

    it("redirects focus back to the container when it tries to escape", async () => {
      render(
        <>
          <button type="button" data-testid="outside">
            Outside
          </button>
          <Modal title="Confirm" onExit={noop}>
            <button type="button">Submit</button>
          </Modal>
        </>
      );

      const modalContainer = document.querySelector(
        ".modal__modal_container"
      ) as HTMLElement;
      await waitFor(() => expect(modalContainer).toHaveFocus());

      // Simulate focus escaping the modal (Tab past the last control, or a
      // stray focus() call landing on background UI).
      fireEvent.focusOut(modalContainer, {
        relatedTarget: screen.getByTestId("outside"),
      });

      expect(modalContainer).toHaveFocus();
    });

    it("does not steal focus on mount when isHidden (stacked modal on top)", () => {
      // When another modal is stacked on top (this modal receives isHidden),
      // that top modal owns focus. This modal must not grab it on mount.
      render(
        <>
          <button type="button" data-testid="stacked">
            Stacked modal control
          </button>
          <Modal title="Bottom" isHidden onExit={noop}>
            <button type="button">Submit</button>
          </Modal>
        </>
      );

      const stacked = screen.getByTestId("stacked");
      stacked.focus();
      expect(stacked).toHaveFocus();
    });

    it("does not refocus on focusout when isHidden", () => {
      // The bottom modal's focusout listener also has to stand down while a
      // stacked modal is on top; otherwise every focus movement inside the
      // top modal would trigger a refocus on the bottom one.
      render(
        <>
          <button type="button" data-testid="stacked">
            Stacked modal control
          </button>
          <Modal title="Bottom" isHidden onExit={noop}>
            <button type="button">Submit</button>
          </Modal>
        </>
      );

      const stacked = screen.getByTestId("stacked");
      const modalContainer = document.querySelector(
        ".modal__modal_container"
      ) as HTMLElement;

      stacked.focus();
      // Focus moving between two elements outside the hidden modal — the
      // hidden modal's listener must not react.
      fireEvent.focusOut(stacked, { relatedTarget: document.body });

      expect(modalContainer).not.toHaveFocus();
    });

    it("does not fire onEnter for autorepeated Enter keys", () => {
      const onEnter = jest.fn();
      render(
        <Modal title="Confirm" onEnter={onEnter} onExit={noop}>
          <div>content</div>
        </Modal>
      );

      // The trigger button's Enter is still down when the modal mounts and
      // this listener attaches; the browser's autorepeat fires more keydowns
      // with event.repeat === true. Those must not fire onEnter.
      fireEvent.keyDown(document, { code: "Enter", repeat: true });
      expect(onEnter).not.toHaveBeenCalled();

      // A fresh Enter (repeat: false) is a real user intent — fires.
      fireEvent.keyDown(document, { code: "Enter", repeat: false });
      expect(onEnter).toHaveBeenCalledTimes(1);
    });

    it("restores focus to the previously focused element on close", async () => {
      const Wrapper = () => {
        const [open, setOpen] = useState(false);
        return (
          <>
            <button
              type="button"
              onClick={() => setOpen(true)}
              data-testid="trigger"
            >
              Open
            </button>
            {open && (
              <Modal title="Confirm" onExit={() => setOpen(false)}>
                <button type="button" onClick={() => setOpen(false)}>
                  Close from inside
                </button>
              </Modal>
            )}
          </>
        );
      };

      const user = userEvent.setup();
      render(<Wrapper />);
      const trigger = screen.getByTestId("trigger");
      trigger.focus();
      expect(trigger).toHaveFocus();

      await user.click(trigger);
      // Container is now focused (not the interior button).
      await waitFor(() => {
        expect(
          screen.getByRole("button", { name: "Close from inside" })
        ).not.toHaveFocus();
      });

      await user.click(
        screen.getByRole("button", { name: "Close from inside" })
      );
      await waitFor(() => {
        expect(
          screen.queryByRole("button", { name: "Close from inside" })
        ).not.toBeInTheDocument();
      });
      await waitFor(() => {
        expect(trigger).toHaveFocus();
      });
    });
  });
});
