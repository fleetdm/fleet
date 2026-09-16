import { renderHook } from "@testing-library/react";

import useBlockNavigation from "./useBlockNavigation";

describe("useBlockNavigation", () => {
  let addSpy: jest.SpyInstance;
  let removeSpy: jest.SpyInstance;

  beforeEach(() => {
    addSpy = jest.spyOn(window, "addEventListener");
    removeSpy = jest.spyOn(window, "removeEventListener");
  });

  afterEach(() => {
    addSpy.mockRestore();
    removeSpy.mockRestore();
  });

  it("attaches a beforeunload handler when block is true", () => {
    renderHook(() => useBlockNavigation(true));
    const beforeunloadAdds = addSpy.mock.calls.filter(
      ([evt]) => evt === "beforeunload"
    );
    expect(beforeunloadAdds).toHaveLength(1);
  });

  it("does not attach a handler when block is false", () => {
    renderHook(() => useBlockNavigation(false));
    const beforeunloadAdds = addSpy.mock.calls.filter(
      ([evt]) => evt === "beforeunload"
    );
    expect(beforeunloadAdds).toHaveLength(0);
  });

  it("removes the handler on unmount when block was true", () => {
    const { unmount } = renderHook(() => useBlockNavigation(true));
    unmount();
    const beforeunloadRemoves = removeSpy.mock.calls.filter(
      ([evt]) => evt === "beforeunload"
    );
    expect(beforeunloadRemoves).toHaveLength(1);
  });

  it("removes the handler when block flips from true to false", () => {
    const { rerender } = renderHook(
      ({ block }: { block: boolean }) => useBlockNavigation(block),
      { initialProps: { block: true } }
    );
    rerender({ block: false });
    const beforeunloadRemoves = removeSpy.mock.calls.filter(
      ([evt]) => evt === "beforeunload"
    );
    expect(beforeunloadRemoves).toHaveLength(1);
  });

  it("preventDefault and returnValue are set by the attached handler", () => {
    renderHook(() => useBlockNavigation(true));
    const handler = addSpy.mock.calls.find(
      ([evt]) => evt === "beforeunload"
    )?.[1] as (e: BeforeUnloadEvent) => void;
    expect(handler).toBeDefined();

    const preventDefault = jest.fn();
    const event = ({
      preventDefault,
      returnValue: false,
    } as unknown) as BeforeUnloadEvent;
    handler(event);
    expect(preventDefault).toHaveBeenCalledTimes(1);
    expect(event.returnValue).toBe(true);
  });
});
