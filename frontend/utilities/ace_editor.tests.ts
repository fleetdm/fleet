import { Ace } from "ace-builds";

import { releaseStuckSelectionOnScroll } from "./ace_editor";

interface IMockMouseHandler {
  isMousePressed: boolean;
  releaseMouse?: jest.Mock;
}

const buildEditor = (mouseHandler: IMockMouseHandler | undefined) => {
  const container = document.createElement("div");
  return {
    editor: ({
      container,
      $mouseHandler: mouseHandler,
    } as unknown) as Ace.Editor,
    container,
  };
};

const scroll = (container: HTMLElement, buttons: number) => {
  container.dispatchEvent(new WheelEvent("wheel", { buttons }));
};

describe("releaseStuckSelectionOnScroll", () => {
  it("releases a stuck selection capture when scrolling with no button pressed", () => {
    const releaseMouse = jest.fn();
    const { editor, container } = buildEditor({
      isMousePressed: true,
      releaseMouse,
    });

    releaseStuckSelectionOnScroll(editor);
    scroll(container, 0);

    expect(releaseMouse).toHaveBeenCalledTimes(1);
  });

  it("does not release while a mouse button is held (a real drag-select)", () => {
    const releaseMouse = jest.fn();
    const { editor, container } = buildEditor({
      isMousePressed: true,
      releaseMouse,
    });

    releaseStuckSelectionOnScroll(editor);
    scroll(container, 1);

    expect(releaseMouse).not.toHaveBeenCalled();
  });

  it("does nothing when the mouse handler is not in a pressed state", () => {
    const releaseMouse = jest.fn();
    const { editor, container } = buildEditor({
      isMousePressed: false,
      releaseMouse,
    });

    releaseStuckSelectionOnScroll(editor);
    scroll(container, 0);

    expect(releaseMouse).not.toHaveBeenCalled();
  });

  it("does not throw when the mouse handler is unavailable", () => {
    const { editor, container } = buildEditor(undefined);

    releaseStuckSelectionOnScroll(editor);

    expect(() => scroll(container, 0)).not.toThrow();
  });
});
