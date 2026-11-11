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
});
