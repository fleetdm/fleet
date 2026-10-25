import React, { useState, useRef, useLayoutEffect } from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";

const MOVED_CLASS = "query-report__column-moved";

interface IProps {
  columns: string[];
  onSave: (columns: string[]) => Promise<void>;
  onExit: () => void;
}

/** Lets users reorder report columns and submit their preferred order. */
const ColumnPreferencesModal = ({ columns, onSave, onExit }: IProps) => {
  const [order, setOrder] = useState(columns);
  const [isSaving, setIsSaving] = useState(false);

  const listRef = useRef<HTMLOListElement>(null);
  const pendingFocus = useRef<{ id: string; offset: number } | null>(null);
  const previousTops = useRef(new Map<string, number>());
  const [announcement, setAnnouncement] = useState("");
  useLayoutEffect(() => {
    const pending = pendingFocus.current;
    if (!pending) return;
    const rows = Array.from(listRef.current?.children || []) as HTMLElement[];
    const row = rows.find(
      (element) => element.getAttribute("data-column-id") === pending.id
    );
    // React reorders rows by moving DOM nodes, which restarts CSS animations,
    // so clear stale highlights before style recalc or earlier rows re-flash.
    rows.forEach((element) => element.classList.remove(MOVED_CLASS));
    if (!window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      // FLIP: start each swapped row at its old position and slide it into place.
      rows.forEach((element) => {
        const previousTop = previousTops.current.get(
          element.getAttribute("data-column-id") || ""
        );
        if (previousTop === undefined) return;
        const delta = previousTop - element.getBoundingClientRect().top;
        if (delta) {
          element.animate?.(
            [{ transform: `translateY(${delta}px)` }, { transform: "none" }],
            { duration: 200, easing: "ease-out" }
          );
        }
      });
      if (row) {
        row.getBoundingClientRect(); // force reflow to restart the highlight
        row.classList.add(MOVED_CLASS);
      }
    }
    previousTops.current.clear();
    const buttons = row?.querySelectorAll<HTMLButtonElement>("button");
    const preferred = buttons?.[pending.offset < 0 ? 0 : 1];
    (preferred?.disabled
      ? buttons?.[pending.offset < 0 ? 1 : 0]
      : preferred
    )?.focus();
    pendingFocus.current = null;
  }, [order]);

  /** Moves a column while preserving focus and announcing its new position. */
  const moveColumn = (index: number, offset: number) => {
    const next = [...order];
    [next[index], next[index + offset]] = [next[index + offset], next[index]];
    pendingFocus.current = { id: order[index], offset };
    Array.from(listRef.current?.children || []).forEach((element) => {
      previousTops.current.set(
        element.getAttribute("data-column-id") || "",
        element.getBoundingClientRect().top
      );
    });
    const label =
      order[index] === "last_fetched" ? "Last fetched" : order[index];
    setAnnouncement(
      `${label} moved to position ${index + offset + 1} of ${order.length}`
    );
    setOrder(next);
  };

  /** Submits the selected order and prevents interaction until saving finishes. */
  const save = async () => {
    setIsSaving(true);
    try {
      await onSave(order);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Modal
      title="Column preferences"
      onExit={onExit}
      disableClosingModal={isSaving}
    >
      <p>
        Choose the column order for this report. These preferences apply only to
        your account.
      </p>
      <ol
        ref={listRef}
        className="query-report__column-preferences"
        aria-label="Column order"
      >
        {order.map((id, index) => {
          const label = id === "last_fetched" ? "Last fetched" : id;
          return (
            <li key={id} data-column-id={id}>
              <span>{label}</span>
              <Button
                variant="secondary"
                size="small"
                ariaLabel={`Move up: ${label}`}
                disabled={isSaving || index === 0}
                onClick={() => moveColumn(index, -1)}
              >
                Move up
              </Button>
              <Button
                variant="secondary"
                size="small"
                ariaLabel={`Move down: ${label}`}
                disabled={isSaving || index === order.length - 1}
                onClick={() => moveColumn(index, 1)}
              >
                Move down
              </Button>
            </li>
          );
        })}
      </ol>
      <p
        className="sr-only"
        role="status"
        aria-label="Column position"
        aria-live="polite"
      >
        {announcement}
      </p>
      <ModalFooter
        primaryButtons={
          <>
            <Button onClick={save} isLoading={isSaving} disabled={isSaving}>
              Save
            </Button>
            <Button variant="secondary" onClick={onExit} disabled={isSaving}>
              Cancel
            </Button>
          </>
        }
      />
    </Modal>
  );
};

export default ColumnPreferencesModal;
