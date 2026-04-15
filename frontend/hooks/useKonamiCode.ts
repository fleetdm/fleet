import { useEffect, useRef } from "react";

/**
 * Classic Konami code sequence: ↑ ↑ ↓ ↓ ← → ← → B A.
 * Stored as lowercase `KeyboardEvent.key` values.
 */
const KONAMI_SEQUENCE = [
  "arrowup",
  "arrowup",
  "arrowdown",
  "arrowdown",
  "arrowleft",
  "arrowright",
  "arrowleft",
  "arrowright",
  "b",
  "a",
];

/**
 * Fires `onUnlock` when the Konami code is typed anywhere in the window.
 * The sequence resets on any mismatched key, so it's forgiving of typos.
 */
const useKonamiCode = (onUnlock: () => void, enabled = true) => {
  const progressRef = useRef(0);
  // Keep the latest callback in a ref so we don't have to reattach the
  // listener every render.
  const callbackRef = useRef(onUnlock);
  callbackRef.current = onUnlock;

  useEffect(() => {
    if (!enabled) {
      progressRef.current = 0;
      return undefined;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      // Ignore key repeats — holding a key shouldn't fill the sequence.
      if (event.repeat) {
        return;
      }

      const expected = KONAMI_SEQUENCE[progressRef.current];
      const pressed = event.key.toLowerCase();

      if (pressed === expected) {
        progressRef.current += 1;
        if (progressRef.current === KONAMI_SEQUENCE.length) {
          progressRef.current = 0;
          callbackRef.current();
        }
        return;
      }

      // Allow the first key of the sequence to restart progress (e.g. extra
      // ArrowUp presses shouldn't abort the combo).
      progressRef.current = pressed === KONAMI_SEQUENCE[0] ? 1 : 0;
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [enabled]);
};

export default useKonamiCode;
