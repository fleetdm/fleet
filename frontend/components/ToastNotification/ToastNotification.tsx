import React from "react";
import { Toaster, toast, ExternalToast } from "sonner";

import ToastCard, { ToastVariant } from "./ToastCard";

const baseClass = "toast-notification";

// Auto close duration (error toast is never closed automatically)
const SUCCESS_DURATION = 5000;
const ERROR_DURATION = Infinity;

// Max number of visible toasts at the same time.
const VISIBLE_TOASTS = 5;

export interface IToastNotificationProps {
  className?: string;
}

const ToastNotification = ({
  className,
}: IToastNotificationProps): JSX.Element => {
  const classes = className
    ? `${baseClass} ${baseClass}__wrapper ${className}`
    : `${baseClass} ${baseClass}__wrapper`;

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

export interface INotifyOptions extends ExternalToast {
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
 * Strip Fleet-specific fields from the options object before forwarding to
 * Sonner. Sonner rejects unknown keys silently, but dropping them keeps the
 * payload clean and debuggable.
 */
const stripFleetOptions = (
  options?: INotifyOptions
): ExternalToast | undefined => {
  if (!options) return undefined;
  const rest = { ...options };
  delete (rest as { response?: unknown }).response;
  delete (rest as { detailLabel?: string }).detailLabel;
  return rest;
};

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
  if (isObject(resp) && "response" in resp && looksLikeResponse(resp.response)) {
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

/**
 * Merge a per-variant default duration with the caller's options. If the
 * caller sets `duration` explicitly (including `Infinity`), that wins.
 */
const withDefaultDuration = (
  variantDuration: number,
  sonnerOptions: ExternalToast | undefined
): ExternalToast => ({
  duration: variantDuration,
  ...(sonnerOptions ?? {}),
});

export const notify: INotify = {
  success: (message, options) =>
    toast.custom(
      (id) => <ToastCard variant="success" message={message} toastId={id} />,
      withDefaultDuration(SUCCESS_DURATION, stripFleetOptions(options))
    ),
  error: (message, options) => {
    const { detail, detailLabel } = resolveDetailProps(options);
    return toast.custom(
      (id) => (
        <ToastCard
          variant="error"
          message={message}
          detail={detail}
          detailLabel={detailLabel}
          toastId={id}
        />
      ),
      withDefaultDuration(ERROR_DURATION, stripFleetOptions(options))
    );
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
