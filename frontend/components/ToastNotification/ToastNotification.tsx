import React, { useEffect } from "react";
import { browserHistory } from "react-router";
import { Toaster, toast, ExternalToast } from "sonner";

import ToastCard, { ToastVariant } from "./ToastCard";

const baseClass = "toast-notification";

// Auto close duration (error toast is never closed automatically)
const SUCCESS_DURATION = 5000;
const ERROR_DURATION = Infinity;

// Fallback copy for error toasts called with an empty message. Error helpers
// (getErrorReason/getErrorMessage) return "" for errors they can't parse — e.g.
// a network error — so without this the toast would render with no message,
// just the raw-response panel. Keeps every error toast meaningful.
const GENERIC_ERROR_MESSAGE = "Something went wrong. Please try again.";

// Max number of visible toasts at the same time.
const VISIBLE_TOASTS = 10;

export interface IToastNotificationProps {
  className?: string;
}

const ToastNotification = ({
  className,
}: IToastNotificationProps): JSX.Element => {
  const classes = className
    ? `${baseClass} ${baseClass}__wrapper ${className}`
    : `${baseClass} ${baseClass}__wrapper`;

  // Dismiss visible toasts on route change, matching 4.86 flash behavior.
  // Only fires when the pathname actually changes — query-param `replace`s
  // (e.g. TableContainer's onQueryChange URL sync on initial render) are
  // ignored so they don't kill a toast that just landed on the destination
  // page (#48180). The listener fires synchronously during router.push;
  // because `notify` defers creation by a tick (see below), a toast
  // triggered alongside a navigation is created afterward and lands on the
  // destination page.
  useEffect(() => {
    let prevPathname = window.location.pathname;
    const unlisten = browserHistory.listen((location) => {
      if (location.pathname !== prevPathname) {
        prevPathname = location.pathname;
        toast.dismiss();
      }
    });
    return unlisten;
  }, []);

  return (
    <Toaster
      className={classes}
      position="bottom-right"
      visibleToasts={VISIBLE_TOASTS}
    />
  );
};

/**
 * Minimal shape of an HTTP response as surfaced by Fleet's API wrapper
 * (`frontend/services/index.ts`). Used by `notify.error(..., { response })`
 * to auto-derive the expandable panel's label and body.
 */
export interface INotifyResponse {
  status?: number;
  statusText?: string;
  data?: unknown;
}

/**
 * Options accepted by `notify.*`. This is Fleet's own curated surface — NOT
 * Sonner's. We expose only the keys we explicitly support and forward; callers
 * should program against this interface rather than reaching for Sonner
 * options. To allow a new Sonner passthrough, add the field here AND copy it
 * through in `toSonnerOptions` below.
 */
export interface INotifyOptions {
  /**
   * Auto-dismiss after N ms. Defaults: 5000 (success), Infinity (error).
   * Forwarded to Sonner.
   */
  duration?: number;
  /**
   * Reuse an id to replace/update an existing toast in place instead of
   * stacking a new one (e.g. a "Saving…" toast that becomes "Saved").
   * Forwarded to Sonner.
   */
  id?: ToastId;
  /**
   * Pass the API response object (or the value a rejected promise
   * yielded from Fleet's `sendRequest` — typically the axios response,
   * which already has the `INotifyResponse` shape). The toast
   * auto-populates the expandable panel with `response.data` as the body
   * and `"Status: {status} {statusText}"` as the label above it.
   *
   * Typed `unknown` so caught errors can be passed directly from
   * `catch (e)` blocks; values without a response shape render as the raw
   * payload with the default label.
   *
   * For non-HTTP payloads, construct a minimal response-shaped object:
   *   `{ response: { data: anyObject } }`.
   */
  response?: unknown;
  /**
   * Override the label shown above the payload. Defaults to the
   * auto-derived `"Status: {status} {statusText}"` when `response` is
   * provided, otherwise `"Raw response"`.
   */
  detailLabel?: string;
}

export type ToastId = string | number;

/**
 * One toast in a `notify.batch` call. Mirrors the single-toast API:
 * `variant` picks `notify.success`/`notify.error` behavior and `options`
 * supports the same fields, including `response` for the expandable
 * raw-response panel on error toasts.
 */
export interface INotifyBatchItem {
  variant: ToastVariant;
  message: React.ReactNode;
  options?: INotifyOptions;
}

/**
 * Imperative API for triggering toasts. Intended to be called from anywhere
 * in the app — handlers, effects, services — once `<ToastNotification />`
 * is mounted at the root.
 */
export interface INotify {
  success: (message: React.ReactNode, options?: INotifyOptions) => ToastId;
  error: (message: React.ReactNode, options?: INotifyOptions) => ToastId;
  /**
   * Render several toasts at once — e.g. one error per failed item in a
   * bulk operation. Items render in array order; the Toaster caps how many
   * are visible at a time (`VISIBLE_TOASTS`), stacking the rest.
   * Returns the ids in the same order as the input for selective `dismiss`.
   */
  batch: (items: INotifyBatchItem[]) => ToastId[];
  dismiss: (id?: ToastId) => void;
}

/**
 * Build the Sonner options object from Fleet's curated `INotifyOptions`,
 * copying through ONLY the keys we choose to forward (the rest of
 * `INotifyOptions` — `response`, `detailLabel` — is consumed by us and never
 * reaches Sonner). The caller's `duration` wins over the per-variant default.
 * To forward a new Sonner option, add it to `INotifyOptions` and copy it here.
 */
const toSonnerOptions = (
  variantDuration: number,
  options?: INotifyOptions
): ExternalToast => ({
  duration: options?.duration ?? variantDuration,
  ...(options?.id !== undefined && { id: options.id }),
});

/**
 * Friendly names for HTTP status codes most commonly returned by Fleet's
 * API, used as a fallback when axios doesn't provide `statusText` (e.g.
 * under HTTP/2 where browsers leave it empty).
 */
const HTTP_STATUS_MEANINGS: Record<number, string> = {
  400: "Bad Request",
  401: "Unauthorized",
  403: "Forbidden",
  404: "Not Found",
  409: "Conflict",
  422: "Unprocessable Entity",
  429: "Too Many Requests",
  500: "Internal Server Error",
  502: "Bad Gateway",
  503: "Service Unavailable",
  504: "Gateway Timeout",
};

/**
 * Derive the body (`detail`) and heading (`detailLabel`) of the expandable
 * panel from the `notify.error` options. `response.data` populates the
 * body; `"Status: {status} {statusText}"` populates the label. An explicit
 * `detailLabel` wins over the auto-derived one.
 */
const isObject = (v: unknown): v is Record<string, unknown> =>
  typeof v === "object" && v !== null;

// A value carries an HTTP response if it exposes a body or a status code.
const looksLikeResponse = (v: unknown): boolean =>
  isObject(v) && ("data" in v || "status" in v);

const resolveDetailProps = (
  options?: INotifyOptions
): { detail: unknown; detailLabel: string | undefined } => {
  if (!options || options.response === undefined) {
    return { detail: undefined, detailLabel: options?.detailLabel };
  }

  // Caught errors are passed as `unknown`. Fleet's `sendRequest` rejects with
  // two shapes: usually the axios *response* (`{ status, data }` at the top
  // level), but for `skipParseError` endpoints (e.g. MDM config) it rejects
  // with the bare AxiosError, whose body lives one level down at
  // `.response.data`. Unwrap that case so the panel reads the real body
  // instead of the AxiosError's (empty) top-level `.data`.
  let resp: unknown = options.response;
  if (
    isObject(resp) &&
    "response" in resp &&
    looksLikeResponse(resp.response)
  ) {
    resp = resp.response;
  }

  if (!looksLikeResponse(resp)) {
    // Non-response payload (e.g. a plain string) — show it raw.
    return { detail: resp, detailLabel: options.detailLabel };
  }

  const fromResponse = resp as INotifyResponse;

  // Prefer the server-supplied status text; fall back to a client-side map
  // of common status codes (statusText is often empty over HTTP/2).
  let autoLabel: string | undefined;
  if (fromResponse.status) {
    const meaning =
      fromResponse.statusText || HTTP_STATUS_MEANINGS[fromResponse.status];
    autoLabel = meaning
      ? `Status: ${fromResponse.status} ${meaning}`
      : `Status: ${fromResponse.status}`;
  }

  return {
    detail: fromResponse.data,
    detailLabel: options.detailLabel ?? autoLabel,
  };
};

// Monotonic id source so we can return a toast id synchronously while
// deferring the actual creation (below). Session-unique is sufficient.
let toastSeq = 0;
const nextToastId = (): ToastId => {
  toastSeq += 1;
  return `fleet-toast-${toastSeq}`;
};

export const notify: INotify = {
  success: (message, options) => {
    const id = options?.id ?? nextToastId();
    // Defer one tick so the toast is created after the route-change
    // dismiss above, landing it on the destination page. When a handler
    // both navigates and shows a success toast, call notify.success
    // before router.push — the reverse order can break auto-dismiss (#48088).
    setTimeout(() => {
      toast.custom(
        (sonnerId) => (
          <ToastCard variant="success" message={message} toastId={sonnerId} />
        ),
        toSonnerOptions(SUCCESS_DURATION, { ...options, id })
      );
    });
    return id;
  },
  error: (message, options) => {
    const id = options?.id ?? nextToastId();
    // Fall back to generic copy when the caller passes an empty/blank message
    // so the toast is never just an icon + raw-response panel.
    const resolvedMessage =
      message === null ||
      message === undefined ||
      (typeof message === "string" && message.trim() === "")
        ? GENERIC_ERROR_MESSAGE
        : message;
    const { detail, detailLabel } = resolveDetailProps(options);
    setTimeout(() => {
      toast.custom(
        (sonnerId) => (
          <ToastCard
            variant="error"
            message={resolvedMessage}
            detail={detail}
            detailLabel={detailLabel}
            toastId={sonnerId}
          />
        ),
        toSonnerOptions(ERROR_DURATION, { ...options, id })
      );
    });
    return id;
  },
  batch: (items) =>
    items.map((item) =>
      item.variant === "success"
        ? notify.success(item.message, item.options)
        : notify.error(item.message, item.options)
    ),
  dismiss: (id) => {
    toast.dismiss(id);
  },
};

export default ToastNotification;
