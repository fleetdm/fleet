import React, { useState } from "react";
import PropTypes from "prop-types";

import Modal from "components/Modal";
import Checkbox from "../../../../../components/forms/fields/Checkbox";
import Button from "../../../../../components/buttons/Button";

const baseClass = "edit-columns-modal";

const useCheckboxListStateManagement = (allColumns, hiddenColumns) => {
  const [columnItems, setColumnItems] = useState(() => {
    return allColumns.map((column) => {
      return {
        name: column.title,
        id: column.id,
        isChecked: !hiddenColumns.includes(column.id),
        disableHidden: column.disableHidden,
      };
    });
  });

  const updateColumnItems = (columnId) => {
    setColumnItems((prevState) => {
      const selectedColumn = columnItems.find(
        (column) => column.id === columnId
      );
      const updatedColumn = {
        ...selectedColumn,
        isChecked: !selectedColumn.isChecked,
      };

      // this is replacing the column object with the updatedColumn we just created.
      const newState = prevState.map((currentColumn) => {
        return currentColumn.id === columnId ? updatedColumn : currentColumn;
      });
      return newState;
    });
  };

  return [columnItems, updateColumnItems];
};

const getHiddenColumns = (columns) => {
  return columns
    .filter((column) => !column.isChecked)
    .map((column) => column.id);
};

const EditColumnsModal = ({
  columns,
  hiddenColumns,
  onSaveColumns,
  onCancelColumns,
}) => {
  const [columnItems, updateColumnItems] = useCheckboxListStateManagement(
    columns,
    hiddenColumns
  );

  return (
    <Modal title="Edit columns" onExit={onCancelColumns} className={baseClass}>
      <div className="form">
        <p>Choose which columns you see:</p>
        <div className={`${baseClass}__column-headers`}>
          {columnItems.map((column) => {
            if (column.disableHidden) return null;
            return (
              <div key={column.id}>
                <Checkbox
                  name={column.name}
                  value={column.isChecked}
                  onChange={() => updateColumnItems(column.id)}
                >
                  <span>{column.name}</span>
                </Checkbox>
              </div>
            );
          })}
        </div>
        <div className="modal-cta-wrap">
          <Button
            onClick={() => onSaveColumns(getHiddenColumns(columnItems))}
            variant="default"
          >
            Save
          </Button>
          <Button onClick={onCancelColumns} variant="inverse">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

EditColumnsModal.propTypes = {
  columns: PropTypes.arrayOf(PropTypes.object), // eslint-disable-line react/forbid-prop-types
  hiddenColumns: PropTypes.arrayOf(PropTypes.string),
  onSaveColumns: PropTypes.func,
  onCancelColumns: PropTypes.func,
};

export default EditColumnsModal;
