import { setThemeMode } from "./theme";

describe("theme - setThemeMode", () => {
  beforeEach(() => {
    document.body.className = "";
    localStorage.clear();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it("does not add theme-transition when picking a mode that resolves to the current state", () => {
    // Body starts without `dark-mode`, i.e. currently light. Picking
    // Light shouldn't trigger the transition class — the App's hero
    // gradient flickers when `transition: background-image` re-runs
    // even between two visually identical gradients.
    setThemeMode("light");

    expect(document.body.classList.contains("theme-transition")).toBe(false);
  });

  it("adds theme-transition when the resolved dark state actually flips", () => {
    // Body starts light → picking Dark is a real transition.
    setThemeMode("dark");

    expect(document.body.classList.contains("theme-transition")).toBe(true);
    expect(document.body.classList.contains("dark-mode")).toBe(true);
  });

  it("removes theme-transition after the 300ms transition window", () => {
    setThemeMode("dark");
    expect(document.body.classList.contains("theme-transition")).toBe(true);

    jest.advanceTimersByTime(300);

    expect(document.body.classList.contains("theme-transition")).toBe(false);
  });

  it("dispatches fleet-theme-change on every applied mode change, including no-op picks", () => {
    // Both the animated path (real flip) and the no-op path (picking
    // a mode that resolves to the current dark state) must dispatch
    // the event, so subscribers like AccountSidePanel can re-read
    // getThemeMode() and keep the radio in sync with the picked mode
    // even when nothing visual changes.
    const handler = jest.fn();
    window.addEventListener("fleet-theme-change", handler);

    setThemeMode("light"); // no-op: body starts without dark-mode
    setThemeMode("dark"); // real flip
    setThemeMode("light"); // real flip back

    expect(handler).toHaveBeenCalledTimes(3);
    expect(handler.mock.calls[0][0].detail).toEqual({ dark: false });
    expect(handler.mock.calls[1][0].detail).toEqual({ dark: true });
    expect(handler.mock.calls[2][0].detail).toEqual({ dark: false });

    window.removeEventListener("fleet-theme-change", handler);
  });

  it("persists explicit modes to localStorage and clears the key for system", () => {
    setThemeMode("dark");
    expect(localStorage.getItem("fleet-theme")).toBe("dark");

    setThemeMode("system");
    expect(localStorage.getItem("fleet-theme")).toBeNull();
  });
});
