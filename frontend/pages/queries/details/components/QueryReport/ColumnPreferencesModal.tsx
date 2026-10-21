import React, { useState, useRef, useLayoutEffect } from "react";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";

interface IProps {
  columns: string[];
  onSave: (columns: string[]) => Promise<void>;
  onExit: () => void;
}

const ColumnPreferencesModal = ({ columns, onSave, onExit }: IProps) => {
  const [order, setOrder] = useState(columns);
  const [isSaving, setIsSaving] = useState(false);

  const listRef = useRef<HTMLOListElement>(null);
  const pendingFocus = useRef<{ id: string; offset: number } | null>(null);
  const [announcement, setAnnouncement] = useState("");
  useLayoutEffect(() => {
    const pending = pendingFocus.current;
    if (!pending) return;
    const row = Array.from(listRef.current?.children || []).find(
      (element) => element.getAttribute("data-column-id") === pending.id
    );
    const buttons = row?.querySelectorAll<HTMLButtonElement>("button");
    const preferred = buttons?.[pending.offset < 0 ? 0 : 1];
    (preferred?.disabled
      ? buttons?.[pending.offset < 0 ? 1 : 0]
      : preferred
    )?.focus();
    pendingFocus.current = null;
  }, [order]);

  const moveColumn = (index: number, offset: number) => {
    const next = [...order];
    [next[index], next[index + offset]] = [next[index + offset], next[index]];
    pendingFocus.current = { id: order[index], offset };
    const label =
      order[index] === "last_fetched" ? "Last fetched" : order[index];
    setAnnouncement(
      `${label} moved to position ${index + offset + 1} of ${order.length}`
    );
    setOrder(next);
  };

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
      <p role="status" aria-label="Column position" aria-live="polite">
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
