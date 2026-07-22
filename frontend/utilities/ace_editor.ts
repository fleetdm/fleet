import { Ace } from "ace-builds";

/**
 * Works around an Ace editor bug where scrolling after a stationary single
 * click selects text instead of scrolling (fleetdm/fleet#48490).
 *
 * Ace's mouse handler keeps an internal "select" capture alive after a
 * mousedown and only tears it down on the next mousemove/mouseup. On a click
 * with no pointer movement (common with trackpads and trackball mice), that
 * capture can stay active. Because Ace's self-healing guard only runs on
 * mousemove, scrolling the editor afterwards keeps re-running its selection
 * logic against the now stationary pointer and extends the selection instead of
 * just scrolling.
 *
 * This releases the stuck capture on wheel events when no mouse button is
 * actually pressed, mirroring Ace's own mousemove guard. The listener uses the
 * capture phase so it runs before Ace processes the scroll, and it lives on
 * editor.container, which is removed when the editor unmounts.
 */
export const releaseStuckSelectionOnScroll = (editor: Ace.Editor): void => {
  // $mouseHandler is not part of Ace's public typings.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const mouseHandler = (editor as any).$mouseHandler;
  editor.container.addEventListener(
    "wheel",
    (e: WheelEvent) => {
      if (mouseHandler?.isMousePressed && !e.buttons) {
        mouseHandler.releaseMouse?.();
      }
    },
    // Capture phase so this runs before Ace processes the scroll; passive since
    // we never call preventDefault (avoids a non-passive wheel-listener warning).
    { capture: true, passive: true }
  );
};
