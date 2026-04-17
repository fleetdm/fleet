import React, { useState } from "react";
import classnames from "classnames";
import { toast } from "sonner";

import Icon from "components/Icon";
import { Colors } from "styles/var/colors";
import { syntaxHighlight } from "utilities/helpers";
import { stringToClipboard } from "utilities/copy_text";

const baseClass = "toast-notification";

export type ToastVariant = "success" | "error";

export interface IToastCardProps {
  /* Success or error. */
  variant: ToastVariant;
  /* Success or error message in the toast. */
  message: string;
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
  const hasDetail = detail !== undefined;
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
  if (hasDetail) {
    try {
      detailText = JSON.stringify(detail, null, 2);
      detailHtml = syntaxHighlight(detail);
    } catch {
      // Circular refs / non-serializable values — fall back to safe text.
      detailText = String(detail);
      detailHtml = detailText
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;");
    }
  }

  const [copied, setCopied] = useState(false);

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

  const handleCopy = (): void => {
    stringToClipboard(copyText)
      .then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      })
      .catch(() => {
        // Clipboard API may reject in insecure contexts — stay silent.
      });
  };

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
            <button
              type="button"
              className={`${baseClass}__action-button`}
              aria-expanded={isOpen}
              aria-controls={panelId}
              aria-label={
                isOpen ? "Collapse error details" : "Expand error details"
              }
              onClick={toggle}
            >
              <span
                className={classnames(`${baseClass}__chevron`, {
                  [`${baseClass}__chevron--open`]: isOpen,
                })}
              >
                <Icon name="chevron-down" color="ui-fleet-black-75" />
              </span>
            </button>
          )}
          <button
            type="button"
            className={`${baseClass}__action-button`}
            aria-label="Dismiss notification"
            onClick={handleClose}
          >
            <Icon name="close" color="ui-fleet-black-75" />
          </button>
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
            <button
              type="button"
              className={`${baseClass}__copy-button`}
              aria-label="Copy raw response to clipboard"
              onClick={handleCopy}
            >
              {copied && (
                <span className={`${baseClass}__copy-confirmation`}>
                  Copied!
                </span>
              )}
              <Icon name="copy" color="ui-fleet-black-75" />
            </button>
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
