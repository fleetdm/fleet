/**
 * Tests the notify imperative API that wraps sonner's toast system.
 * Covers: success/error creation, empty-message fallback, batch,
 * dismiss, and response detail resolution.
 */
import { toast } from "sonner";

import { notify } from "./ToastNotification";

jest.mock("sonner", () => ({
  toast: {
    custom: jest.fn(),
    dismiss: jest.fn(),
  },
  Toaster: () => null,
}));

const mockedToast = jest.mocked(toast);

describe("notify - sonner toast API", () => {
  beforeEach(() => {
    jest.useFakeTimers();
    mockedToast.custom.mockClear();
    mockedToast.dismiss.mockClear();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it("notify.success returns a toast id and calls toast.custom with correct options", () => {
    const id = notify.success("Saved!");
    expect(typeof id).toBe("string");
    expect(id).toMatch(/^fleet-toast-/);

    jest.runAllTimers();
    expect(mockedToast.custom).toHaveBeenCalledTimes(1);

    const [, options] = mockedToast.custom.mock.calls[0];
    expect(options).toMatchObject({ duration: 5000, id });
  });

  it("notify.error returns a toast id and calls toast.custom with infinite duration", () => {
    const id = notify.error("Something failed");
    jest.runAllTimers();
    expect(typeof id).toBe("string");
    expect(mockedToast.custom).toHaveBeenCalledTimes(1);

    const [, options] = mockedToast.custom.mock.calls[0];
    expect(options).toMatchObject({ duration: Infinity, id });
  });

  it("notify.error with empty message uses generic fallback", () => {
    notify.error("");
    jest.runAllTimers();

    const renderFn = mockedToast.custom.mock.calls[0][0];
    const element = renderFn("test-id");
    expect(element.props.message).toBe(
      "Something went wrong. Please try again."
    );
  });

  it("notify.error with null message uses generic fallback", () => {
    notify.error(null);
    jest.runAllTimers();

    const renderFn = mockedToast.custom.mock.calls[0][0];
    const element = renderFn("test-id");
    expect(element.props.message).toBe(
      "Something went wrong. Please try again."
    );
  });

  it("notify.success with custom id reuses that id", () => {
    const id = notify.success("Updated", { id: "my-custom-id" });
    expect(id).toBe("my-custom-id");

    jest.runAllTimers();
    const [, options] = mockedToast.custom.mock.calls[0];
    expect(options.id).toBe("my-custom-id");
  });

  it("notify.dismiss calls toast.dismiss", () => {
    const id = notify.success("temp");
    notify.dismiss(id);
    expect(mockedToast.dismiss).toHaveBeenCalledWith(id);
  });

  it("notify.batch creates multiple toasts and returns ids", () => {
    const ids = notify.batch([
      { variant: "success", message: "Created host" },
      { variant: "error", message: "Failed to create policy" },
      { variant: "success", message: "Updated config" },
    ]);

    expect(ids).toHaveLength(3);
    ids.forEach((id) => expect(typeof id).toBe("string"));

    jest.runAllTimers();
    expect(mockedToast.custom).toHaveBeenCalledTimes(3);
  });

  it("notify.error with response auto-derives status label", () => {
    notify.error("API error", {
      response: { status: 422, statusText: "", data: { error: "bad input" } },
    });
    jest.runAllTimers();

    const renderFn = mockedToast.custom.mock.calls[0][0];
    const element = renderFn("test-id");
    expect(element.props.detailLabel).toBe(
      "Status: 422 Unprocessable Entity"
    );
    expect(element.props.detail).toEqual({ error: "bad input" });
  });

  it("notify.error with nested response unwraps correctly", () => {
    notify.error("Request failed", {
      response: {
        response: { status: 500, data: { message: "internal" } },
      },
    });
    jest.runAllTimers();

    const renderFn = mockedToast.custom.mock.calls[0][0];
    const element = renderFn("test-id");
    expect(element.props.detail).toEqual({ message: "internal" });
  });
});
