import React from "react";
import { Toaster, toast, ExternalToast } from "sonner";

import ToastCard from "./ToastCard";

const baseClass = "toast-notification";

/**
 * Per-variant auto-dismiss duration (ms). Success toasts fade out after a
 * few seconds; error toasts stay until the user clicks the close button
 * (Sonner interprets `Infinity` as "never auto-close").
 */
const SUCCESS_DURATION = 10000;
const ERROR_DURATION = Infinity;

/**
 * Max number of visible toasts rendered by Sonner at the same time.
 */
const VISIBLE_TOASTS = 3;

export interface IToastNotificationProps {
  className?: string;
}

/**
 * `<ToastNotification />` is a pre-configured Sonner `<Toaster />` wrapper.
 *
 * Mount this component once near the root of the app to enable imperative
 * toast notifications via the exported `notify` helper.
 *
 * We use Sonner in headless mode: every toast is rendered via
 * `toast.custom()` with our own `<ToastCard />` UI, so the visuals do not
 * depend on Sonner's built-in styled variants.
 *
 * NOTE: Additive only — it does not replace the existing `FlashMessage`
 * component.
 */
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
   * yielded from Fleet's `sendRequest`). The toast auto-populates the
   * expandable panel with `response.data` as the body and `"Status:
   * {status} {statusText}"` as the label above it.
   *
   * For non-HTTP payloads, construct a minimal response-shaped object:
   *   `{ response: { data: anyObject } }`.
   */
  response?: INotifyResponse;
  /**
   * Override the label shown above the payload. Defaults to the
   * auto-derived `"Status: {status} {statusText}"` when `response` is
   * provided, otherwise `"Raw response"`.
   */
  detailLabel?: string;
}

export type ToastId = string | number;

/**
 * Imperative API for triggering toasts. Intended to be called from anywhere
 * in the app — handlers, effects, services — once `<ToastNotification />`
 * is mounted at the root.
 */
export interface INotify {
  success: (message: string, options?: INotifyOptions) => ToastId;
  error: (message: string, options?: INotifyOptions) => ToastId;
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
  delete (rest as { response?: INotifyResponse }).response;
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
const resolveDetailProps = (
  options?: INotifyOptions
): { detail: unknown; detailLabel: string | undefined } => {
  if (!options) return { detail: undefined, detailLabel: undefined };

  const fromResponse = options.response;

  // Prefer the server-supplied status text; fall back to a client-side map
  // of common status codes (statusText is often empty over HTTP/2).
  let autoLabel: string | undefined;
  if (fromResponse?.status) {
    const meaning =
      fromResponse.statusText || HTTP_STATUS_MEANINGS[fromResponse.status];
    autoLabel = meaning
      ? `Status: ${fromResponse.status} ${meaning}`
      : `Status: ${fromResponse.status}`;
  }

  return {
    detail: fromResponse?.data,
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
  dismiss: (id) => {
    toast.dismiss(id);
  },
};

export default ToastNotification;
