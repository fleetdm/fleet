import React, { useState } from "react";
import classnames from "classnames";
import { toast } from "sonner";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import CopyButton from "components/buttons/CopyButton";
import { Colors } from "styles/var/colors";
import { syntaxHighlight } from "utilities/helpers";

const baseClass = "toast-notification";

export type ToastVariant = "success" | "error";

export interface IToastCardProps {
  /* Success or error. */
  variant: ToastVariant;
  /* Success or error message in the toast. Accepts JSX for rich
  formatting (e.g. bolded entity names). */
  message: React.ReactNode;
  /**
   * Optional raw payload (e.g. API error response). When provided on an
   * error toast, the card renders a chevron toggle that reveals a formatted
   * JSON panel below the message.
   */
  detail?: unknown;
  /**
   * Label shown above the error response (detail). Defaults to "Raw response".
   */
  detailLabel?: string;
  toastId: string | number;
}

const variantIcon: Record<
  ToastVariant,
  { name: "success-outline" | "error-outline"; color: Colors }
> = {
  success: { name: "success-outline", color: "status-success" },
  error: { name: "error-outline", color: "status-error" },
};

// Serialized payloads that hold nothing worth revealing. Compared against the
// pretty-printed JSON, so an empty object is "{}" rather than "{\n}".
const EMPTY_DETAIL_TEXT = ["", "{}", "[]", "null", '""'];

/**
 * `ToastCard` is the single source of truth for every toast variant. It is
 * rendered inside Sonner's headless `toast.custom()` wrapper — Sonner
 * provides positioning, stacking, and lifecycle; every pixel of the card
 * itself (surface, icon, actions, expandable panel) is ours, so the design
 * does not depend on Sonner's built-in themes.
 *
 * Internal-only — not exported from `./index.ts`.
 */
const ToastCard = ({
  variant,
  message,
  detail,
  detailLabel = "Raw response",
  toastId,
}: IToastCardProps): JSX.Element => {
  const [isOpen, setIsOpen] = useState(false);
  const icon = variantIcon[variant];

  const toggle = (): void => {
    setIsOpen((prev) => !prev);
  };

  const handleClose = (): void => {
    toast.dismiss(toastId);
  };

  // Fleet's shared helper stringifies + escapes + wraps tokens in
  // `<span class="string|number|boolean|null|key">`. The global `pre`
  // rule in `styles/global/_global.scss` then colours each class —
  // identical to the "Manage activity automations" modal's payload.
  let detailHtml = "";
  let detailText = "";
  if (detail !== undefined) {
    try {
      // Values with no JSON representation (functions, symbols) return
      // undefined here rather than throwing, so treat them as empty and skip
      // the highlighter, which assumes a string.
      detailText = JSON.stringify(detail, null, 2) ?? "";
      if (detailText !== "") {
        detailHtml = syntaxHighlight(detail);
      }
    } catch {
      // Circular refs / non-serializable values — fall back to safe text.
      detailText = String(detail);
      detailHtml = detailText
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;");
    }
  }

  // Callers pass caught errors straight through as `response`, and an `Error`'s
  // own properties (message, stack) are non-enumerable, so JSON.stringify
  // returns "{}" without throwing — the panel would open on an empty object.
  // Treat a payload that carries nothing as no payload at all: the message
  // already holds the error text, and an empty panel is worse than no panel.
  const hasDetail = !EMPTY_DETAIL_TEXT.includes(detailText);

  // Capture when the toast first rendered. Snapshotted once via the lazy
  // initializer so the timestamp stays stable across re-renders (toggling
  // the panel, clicking copy, etc.). Not shown in the UI — only included
  // in the clipboard payload for reporting / pasting into tickets.
  const [timestamp] = useState(() => new Date().toISOString());

  // Composed clipboard payload:
  //   Status: 409 Conflict           ← detailLabel (if set)
  //   Timestamp: 2026-04-15T…Z       ← when the toast fired
  //   <blank line>
  //   { ...pretty-printed JSON... }
  const copyText = [detailLabel, `Timestamp: ${timestamp}`, "", detailText]
    .filter((line) => line !== undefined)
    .join("\n");

  const panelId = `${baseClass}__panel-${toastId}`;

  return (
    <div
      className={classnames(
        `${baseClass}__card`,
        `${baseClass}__card--${variant}`,
        {
          [`${baseClass}__card--open`]: hasDetail && isOpen,
        }
      )}
      role="alert"
    >
      <div className={`${baseClass}__header`}>
        <div className={`${baseClass}__icon-message`}>
          <span className={`${baseClass}__icon`}>
            <Icon name={icon.name} color={icon.color} />
          </span>
          <span className={`${baseClass}__message`}>{message}</span>
        </div>
        <div className={`${baseClass}__actions`}>
          {hasDetail && (
            <Button
              className={classnames(`${baseClass}__chevron`, {
                [`${baseClass}__chevron--open`]: isOpen,
              })}
              variant="subdued"
              icon="chevron-down"
              ariaExpanded={isOpen}
              ariaControls={panelId}
              ariaLabel={
                isOpen ? "Collapse error details" : "Expand error details"
              }
              onClick={toggle}
            />
          )}
          <Button
            variant="subdued"
            icon="close"
            ariaLabel="Dismiss notification"
            onClick={handleClose}
          />
        </div>
      </div>
      {hasDetail && isOpen && (
        <div
          id={panelId}
          className={`${baseClass}__panel`}
          role="region"
          aria-label="Error details"
        >
          <div className={`${baseClass}__panel-header`}>
            <span className={`${baseClass}__panel-label`}>{detailLabel}</span>
            <CopyButton
              copyText={copyText}
              size="small"
              ariaLabel="Copy raw response to clipboard"
            />
          </div>
          <pre
            className={`${baseClass}__json-block`}
            // eslint-disable-next-line react/no-danger
            dangerouslySetInnerHTML={{ __html: detailHtml }}
          />
        </div>
      )}
    </div>
  );
};

export default ToastCard;
