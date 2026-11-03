import { act, renderHook } from "@testing-library/react";

import autofillAPI, {
  IAutofillConfigProfileContentResponse,
  IAutofillConfigProfileNameDescriptionResponse,
} from "services/entities/autofill";

import useProfileGeneration, {
  GENERATION_TIMEOUT_MS,
  READING_STEP_MS,
  toGenerationError,
} from "./useProfileGeneration";

const PROFILE: IAutofillConfigProfileContentResponse = {
  type: "mobileconfig",
  name: "AirDrop Disabled",
  content: "<plist/>",
  delivery_notes: "",
};

const NAME_DESCRIPTION: IAutofillConfigProfileNameDescriptionResponse = {
  name: "Disable AirDrop",
  description: "Turns off AirDrop.",
};

const REQUEST = {
  instructions: "Turn off AirDrop",
  platform: "apple" as const,
};

/** A promise the test settles by hand, recording the signal it was sent. */
interface IDeferred<T> {
  promise: Promise<T>;
  resolve: (value: T) => void;
  reject: (reason: unknown) => void;
}

function deferred<T>(): IDeferred<T> {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

const apiError = (name: string, reason: string) => ({
  status: 422,
  data: { message: "Couldn't generate", errors: [{ name, reason }] },
});

describe("useProfileGeneration", () => {
  let nameDescription: IDeferred<IAutofillConfigProfileNameDescriptionResponse>;
  let content: IDeferred<IAutofillConfigProfileContentResponse>;
  let nameSpy: jest.SpyInstance;
  let contentSpy: jest.SpyInstance;

  beforeEach(() => {
    jest.useFakeTimers();
    nameDescription = deferred();
    content = deferred();
    nameSpy = jest
      .spyOn(autofillAPI, "getConfigProfileNameDescription")
      .mockReturnValue(nameDescription.promise);
    contentSpy = jest
      .spyOn(autofillAPI, "getConfigProfileContent")
      .mockReturnValue(content.promise);
  });

  afterEach(() => {
    jest.useRealTimers();
    jest.restoreAllMocks();
  });

  const start = (onTimeout = jest.fn()) => {
    const hook = renderHook(() => useProfileGeneration({ onTimeout }));
    act(() => hook.result.current.generate({ ...REQUEST, fleetId: 2 }));
    return hook;
  };

  it("sends both requests at once, with the fleet only on the content request", () => {
    start();
    expect(nameSpy).toHaveBeenCalledWith(
      expect.objectContaining({ ...REQUEST, signal: expect.anything() })
    );
    expect(nameSpy.mock.calls[0][0]).not.toHaveProperty("fleetId");
    expect(contentSpy).toHaveBeenCalledWith(
      expect.objectContaining({ ...REQUEST, fleetId: 2 })
    );
  });

  it("holds Reading request, then drafts until the name arrives, then writes", async () => {
    const { result } = start();
    expect(result.current.status).toBe("generating");
    expect(result.current.step).toBe("reading");

    // a name that arrives during the hold doesn't skip it
    await act(async () => nameDescription.resolve(NAME_DESCRIPTION));
    expect(result.current.step).toBe("reading");

    act(() => jest.advanceTimersByTime(READING_STEP_MS));
    expect(result.current.step).toBe("writing");
    expect(result.current.nameDescription).toEqual(NAME_DESCRIPTION);
  });

  it("drafts while the name is still pending after the hold", () => {
    const { result } = start();
    act(() => jest.advanceTimersByTime(READING_STEP_MS));
    expect(result.current.step).toBe("drafting");
  });

  it("finishes on the profile without waiting for the name, which can still arrive", async () => {
    const { result } = start();
    await act(async () => content.resolve(PROFILE));
    expect(result.current.status).toBe("done");
    expect(result.current.profile).toEqual(PROFILE);
    expect(result.current.nameDescription).toBeUndefined();

    await act(async () => nameDescription.resolve(NAME_DESCRIPTION));
    expect(result.current.nameDescription).toEqual(NAME_DESCRIPTION);
  });

  it("records a failed name request as null and still finishes", async () => {
    const { result } = start();
    await act(async () => nameDescription.reject(apiError("base", "x")));
    expect(result.current.nameDescription).toBeNull();
    await act(async () => content.resolve(PROFILE));
    expect(result.current.status).toBe("done");
  });

  it("reports a decline on the instructions and discards the name shown", async () => {
    const { result } = start();
    await act(async () => nameDescription.resolve(NAME_DESCRIPTION));
    await act(async () =>
      content.reject(
        apiError(
          "natural_language_instructions",
          "AirDrop can't be set on Windows."
        )
      )
    );
    expect(result.current.status).toBe("error");
    expect(result.current.error).toEqual({
      kind: "decline",
      reason: "AirDrop can't be set on Windows.",
    });
    expect(result.current.nameDescription).toBeUndefined();
  });

  it("reports any other content error as a failure with its reason", async () => {
    const { result } = start();
    await act(async () =>
      content.reject(apiError("base", "The AI service is unavailable."))
    );
    expect(result.current.error).toEqual({
      kind: "failure",
      reason: "The AI service is unavailable.",
    });
  });

  it("treats Fleet's own AI service timeout as the cap", async () => {
    const onTimeout = jest.fn();
    const { result } = start(onTimeout);
    await act(async () =>
      content.reject(apiError("base", "The AI service timed out."))
    );
    expect(onTimeout).toHaveBeenCalledTimes(1);
    expect(nameSpy.mock.calls[0][0].signal.aborted).toBe(true);
    expect(result.current.status).toBe("idle");
    expect(result.current.error).toBeUndefined();
    // the cap doesn't fire a second time
    act(() => jest.advanceTimersByTime(GENERATION_TIMEOUT_MS));
    expect(onTimeout).toHaveBeenCalledTimes(1);
  });

  it("aborts both requests on Stop and returns to idle", () => {
    const { result } = start();
    const { signal } = contentSpy.mock.calls[0][0];
    act(() => result.current.stop());
    expect(signal.aborted).toBe(true);
    expect(nameSpy.mock.calls[0][0].signal.aborted).toBe(true);
    expect(result.current.status).toBe("idle");
  });

  it("aborts both requests at the cap and calls onTimeout", () => {
    const onTimeout = jest.fn();
    const { result } = start(onTimeout);
    act(() => jest.advanceTimersByTime(GENERATION_TIMEOUT_MS - 1));
    expect(onTimeout).not.toHaveBeenCalled();
    act(() => jest.advanceTimersByTime(1));
    expect(onTimeout).toHaveBeenCalledTimes(1);
    expect(contentSpy.mock.calls[0][0].signal.aborted).toBe(true);
    expect(result.current.status).toBe("idle");
  });

  it("doesn't time out once the profile has arrived", async () => {
    const onTimeout = jest.fn();
    start(onTimeout);
    await act(async () => content.resolve(PROFILE));
    act(() => jest.advanceTimersByTime(GENERATION_TIMEOUT_MS));
    expect(onTimeout).not.toHaveBeenCalled();
  });

  it("ignores the previous run's responses after a retry", async () => {
    const { result } = start();
    const second = deferred<IAutofillConfigProfileContentResponse>();
    contentSpy.mockReturnValue(second.promise);
    nameSpy.mockReturnValue(new Promise(() => undefined));
    act(() => result.current.generate(REQUEST));

    await act(async () => content.resolve(PROFILE));
    expect(result.current.status).toBe("generating");

    await act(async () =>
      second.resolve({ ...PROFILE, name: "Second", type: "mobileconfig" })
    );
    expect(result.current.profile?.name).toBe("Second");
  });

  it("aborts the requests when unmounted", () => {
    const { unmount } = start();
    unmount();
    expect(contentSpy.mock.calls[0][0].signal.aborted).toBe(true);
  });
});

describe("toGenerationError", () => {
  it("falls back to an empty reason when the response has none", () => {
    expect(toGenerationError("Network Error")).toEqual({
      kind: "failure",
      reason: "",
    });
  });
});
