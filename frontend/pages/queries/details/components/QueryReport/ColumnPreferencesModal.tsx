import React, { useState } from "react";
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

  const moveColumn = (index: number, offset: number) => {
    const next = [...order];
    [next[index], next[index + offset]] = [next[index + offset], next[index]];
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
        className="query-report__column-preferences"
        aria-label="Column order"
      >
        {order.map((id, index) => {
          const label = id === "last_fetched" ? "Last fetched" : id;
          return (
            <li key={id}>
              <span>{label}</span>
              <Button
                variant="secondary"
                size="small"
                ariaLabel={`Move ${label} left`}
                disabled={isSaving || index === 0}
                onClick={() => moveColumn(index, -1)}
              >
                Move left
              </Button>
              <Button
                variant="secondary"
                size="small"
                ariaLabel={`Move ${label} right`}
                disabled={isSaving || index === order.length - 1}
                onClick={() => moveColumn(index, 1)}
              >
                Move right
              </Button>
            </li>
          );
        })}
      </ol>
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
