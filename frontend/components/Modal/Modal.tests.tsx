import { render, screen, fireEvent, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { noop } from "lodash";
import React, { useState } from "react";

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

  describe("page scroll lock", () => {
    const PAGE_SCROLLBAR_WIDTH = 15;
    let clientWidthSpy: jest.SpyInstance;

    const getBodyPaddingRight = () => document.body.style.paddingRight;

    beforeEach(() => {
      clientWidthSpy = jest
        .spyOn(document.documentElement, "clientWidth", "get")
        .mockReturnValue(window.innerWidth - PAGE_SCROLLBAR_WIDTH);
    });
    afterEach(() => clientWidthSpy.mockRestore());

    it("locks page scroll and pads the page by its scrollbar's width until the modal closes", () => {
      const ClosableModal = () => {
        const [isOpen, setIsOpen] = useState(true);
        return isOpen ? (
          <Modal title="Test" onExit={() => setIsOpen(false)}>
            <div>content</div>
          </Modal>
        ) : null;
      };
      render(<ClosableModal />);

      expect(document.body).toHaveClass("modal-open");
      expect(getBodyPaddingRight()).toBe(`${PAGE_SCROLLBAR_WIDTH}px`);

      fireEvent.keyDown(document, { key: "Escape" });
      act(() => jest.runAllTimers());

      expect(screen.queryByText("content")).not.toBeInTheDocument();
      expect(document.body).not.toHaveClass("modal-open");
      expect(getBodyPaddingRight()).toBe("");
    });

    it("adds no padding when the page has no scrollbar", () => {
      clientWidthSpy.mockReturnValue(window.innerWidth);
      render(
        <Modal title="Test" onExit={noop}>
          <div>content</div>
        </Modal>
      );

      expect(document.body).toHaveClass("modal-open");
      expect(getBodyPaddingRight()).toBe("0px");
    });

    it("keeps the first modal's padding while a second modal is stacked on it", () => {
      const firstModal = render(
        <Modal title="First" onExit={noop}>
          <div>first</div>
        </Modal>
      );
      // The first modal already hid the page scrollbar.
      clientWidthSpy.mockReturnValue(window.innerWidth);
      const secondModal = render(
        <Modal title="Second" onExit={noop}>
          <div>second</div>
        </Modal>
      );
      expect(getBodyPaddingRight()).toBe(`${PAGE_SCROLLBAR_WIDTH}px`);

      secondModal.unmount();
      expect(document.body).toHaveClass("modal-open");
      expect(getBodyPaddingRight()).toBe(`${PAGE_SCROLLBAR_WIDTH}px`);

      firstModal.unmount();
      expect(document.body).not.toHaveClass("modal-open");
      expect(getBodyPaddingRight()).toBe("");
    });
  });

  describe("focus trap", () => {
    it("sets role=dialog and aria-modal=true for assistive tech", () => {
      render(
        <Modal title="Accessible" onExit={noop}>
          <div>content</div>
        </Modal>
      );

      const dialog = screen.getByRole("dialog", { name: "Accessible" });
      expect(dialog).toHaveAttribute("aria-modal", "true");
    });

    it("gives JSX titles an accessible name via aria-labelledby", () => {
      render(
        <Modal title={<em>Fancy title</em>} onExit={noop}>
          <div>content</div>
        </Modal>
      );

      expect(
        screen.getByRole("dialog", { name: "Fancy title" })
      ).toBeInTheDocument();
    });

    it("focuses the first focusable element on open", () => {
      render(
        <Modal title="Focus" onExit={noop}>
          <input type="text" aria-label="first-input" />
        </Modal>
      );

      // Close button in the header is the first focusable in DOM order.
      expect(screen.getByTestId("close-icon").closest("button")).toHaveFocus();
    });

    it("restores focus to the trigger element when the modal unmounts", () => {
      const trigger = document.createElement("button");
      trigger.textContent = "open";
      document.body.appendChild(trigger);
      trigger.focus();

      const { unmount } = render(
        <Modal title="Focus restore" onExit={noop}>
          <input type="text" aria-label="inside" />
        </Modal>
      );

      unmount();

      expect(trigger).toHaveFocus();
      document.body.removeChild(trigger);
    });

    it("traps Tab from the last focusable back to the first", () => {
      render(
        <Modal title="Trap" onExit={noop}>
          <input type="text" aria-label="only-input" />
        </Modal>
      );

      const closeButton = screen.getByTestId("close-icon").closest("button");
      const input = screen.getByLabelText("only-input");

      input.focus();
      fireEvent.keyDown(document, { key: "Tab" });
      expect(closeButton).toHaveFocus();
    });

    it("traps Shift+Tab from the first focusable back to the last", () => {
      render(
        <Modal title="Trap" onExit={noop}>
          <input type="text" aria-label="only-input" />
        </Modal>
      );

      const closeButton = screen.getByTestId("close-icon").closest("button");
      const input = screen.getByLabelText("only-input");

      closeButton?.focus();
      fireEvent.keyDown(document, { key: "Tab", shiftKey: true });
      expect(input).toHaveFocus();
    });

    it("pulls focus back into the modal when Tab is pressed from outside", () => {
      const outside = document.createElement("button");
      outside.textContent = "outside";
      document.body.appendChild(outside);

      render(
        <Modal title="Trap" onExit={noop}>
          <input type="text" aria-label="inside" />
        </Modal>
      );

      outside.focus();
      fireEvent.keyDown(document, { key: "Tab" });
      const closeButton = screen.getByTestId("close-icon").closest("button");
      expect(closeButton).toHaveFocus();
      document.body.removeChild(outside);
    });

    it("treats focus on the dialog container as a backward boundary for Shift+Tab", () => {
      const outside = document.createElement("button");
      outside.textContent = "outside";
      document.body.appendChild(outside);
      outside.focus();

      const { container } = render(
        <Modal title="Trap" onExit={noop}>
          <input type="text" aria-label="inside" />
        </Modal>
      );
      const dialog = container.querySelector<HTMLElement>('[role="dialog"]');
      const input = screen.getByLabelText("inside");
      dialog?.focus();

      fireEvent.keyDown(document, { key: "Tab", shiftKey: true });
      expect(input).toHaveFocus();
      document.body.removeChild(outside);
    });

    it("skips trailing controls with tabIndex=-1 when wrapping Tab", () => {
      render(
        <Modal title="Trap" onExit={noop}>
          <input type="text" aria-label="real-input" />
          <button type="button" tabIndex={-1}>
            skipped-trailing
          </button>
        </Modal>
      );

      const realInput = screen.getByLabelText("real-input");
      realInput.focus();
      fireEvent.keyDown(document, { key: "Tab" });

      // Must wrap to the close button, not land on the tabIndex=-1 button.
      expect(screen.getByTestId("close-icon").closest("button")).toHaveFocus();
    });

    it("skips controls inside display:none ancestors when choosing boundaries", () => {
      render(
        <Modal title="Trap" onExit={noop}>
          <input type="text" aria-label="real-input" />
          <div style={{ display: "none" }}>
            <input type="text" aria-label="hidden-input" />
          </div>
        </Modal>
      );

      const closeButton = screen.getByTestId("close-icon").closest("button");
      closeButton?.focus();
      fireEvent.keyDown(document, { key: "Tab", shiftKey: true });

      // Must land on the real input, not the display:none one.
      expect(screen.getByLabelText("real-input")).toHaveFocus();
    });

    it("does not steal focus from an autofocused child on open", () => {
      const AutoFocusInput = () => {
        const ref = React.useRef<HTMLInputElement>(null);
        React.useEffect(() => {
          ref.current?.focus();
        }, []);
        return <input ref={ref} type="text" aria-label="auto" />;
      };
      render(
        <Modal title="AutoFocus" onExit={noop}>
          <AutoFocusInput />
        </Modal>
      );
      expect(screen.getByLabelText("auto")).toHaveFocus();
    });

    it("restores focus to the opener even when a child autofocuses on mount", () => {
      const trigger = document.createElement("button");
      trigger.textContent = "open";
      document.body.appendChild(trigger);
      trigger.focus();

      const AutoFocusInput = () => {
        const ref = React.useRef<HTMLInputElement>(null);
        React.useEffect(() => {
          ref.current?.focus();
        }, []);
        return <input ref={ref} type="text" aria-label="auto" />;
      };
      const { unmount } = render(
        <Modal title="Restore" onExit={noop}>
          <AutoFocusInput />
        </Modal>
      );
      unmount();

      expect(trigger).toHaveFocus();
      document.body.removeChild(trigger);
    });

    it("moves focus into a dialog that transitions from hidden to visible", () => {
      const Stacked = () => {
        const [showSecond, setShowSecond] = useState(false);
        return (
          <>
            <Modal title="First" isHidden={showSecond} onExit={noop}>
              <button type="button" onClick={() => setShowSecond(true)}>
                open-second
              </button>
            </Modal>
            {showSecond && (
              <Modal title="Second" onExit={noop}>
                <input type="text" aria-label="second-input" />
              </Modal>
            )}
          </>
        );
      };
      render(<Stacked />);

      fireEvent.click(screen.getByText("open-second"));

      const secondCloseButton = screen
        .getAllByTestId("close-icon")[1]
        .closest("button");
      expect(secondCloseButton).toHaveFocus();
    });

    it("yields Tab handling to a higher modal dialog above it", () => {
      render(
        <Modal title="Lower" onExit={noop}>
          <input type="text" aria-label="lower-input" />
        </Modal>
      );

      // Simulate the command palette (cmdk wraps Radix Dialog) opening above.
      // Its root sets role="dialog" and aria-modal="true" but doesn't join
      // Fleet Modal's focusTrapStack.
      const palette = document.createElement("div");
      palette.setAttribute("role", "dialog");
      palette.setAttribute("aria-modal", "true");
      const paletteInput = document.createElement("input");
      palette.appendChild(paletteInput);
      document.body.appendChild(palette);
      paletteInput.focus();

      fireEvent.keyDown(document, { key: "Tab" });

      // Focus must stay in the palette; the modal below must not pull it back.
      expect(paletteInput).toHaveFocus();
      document.body.removeChild(palette);
    });

    it("only the topmost of two stacked modals handles Tab", () => {
      render(
        <>
          <Modal title="Lower" onExit={noop}>
            <input type="text" aria-label="lower-input" />
          </Modal>
          <Modal title="Upper" onExit={noop}>
            <input type="text" aria-label="upper-input" />
          </Modal>
        </>
      );

      const upperInput = screen.getByLabelText("upper-input");
      upperInput.focus();
      fireEvent.keyDown(document, { key: "Tab" });

      // Upper is topmost; Tab from its last focusable (the input) wraps back
      // to its first focusable (its own close button), not the lower modal.
      const upperCloseButton = screen
        .getAllByTestId("close-icon")[1]
        .closest("button");
      expect(upperCloseButton).toHaveFocus();
    });
  });
});
