import React, { useState } from "react";
import PropTypes from "prop-types";

import Checkbox from "../../../../../components/forms/fields/Checkbox";
import Button from "../../../../../components/buttons/Button";

const useCheckboxListStateManagement = (allColumns, hiddenColumns) => {
  const [columnItems, setColumnItems] = useState(() => {
    return allColumns.map((column) => {
      return {
        name: column.title,
        accessor: column.accessor,
        isChecked: !hiddenColumns.includes(column.accessor),
        disableHidden: column.disableHidden,
      };
    });
  });

  const updateColumnItems = (columnAccessor) => {
    setColumnItems((prevState) => {
      const selectedColumn = columnItems.find(
        (column) => column.accessor === columnAccessor
      );
      const updatedColumn = {
        ...selectedColumn,
        isChecked: !selectedColumn.isChecked,
      };

      // this is replacing the column object with the updatedColumn we just created.
      const newState = prevState.map((currentColumn) => {
        return currentColumn.accessor === columnAccessor
          ? updatedColumn
          : currentColumn;
      });
      return newState;
    });
  };

  return [columnItems, updateColumnItems];
};

const getHiddenColumns = (columns) => {
  return columns
    .filter((column) => !column.isChecked)
    .map((column) => column.accessor);
};

const EditColumnsModal = (props) => {
  const { columns, hiddenColumns, onSaveColumns, onCancelColumns } = props;

  const [columnItems, updateColumnItems] = useCheckboxListStateManagement(
    columns,
    hiddenColumns
  );

  return (
    <div className={"edit-column-modal"}>
      <p>Choose which columns you see</p>
      <div className={"modal-items"}>
        {columnItems.map((column) => {
          if (column.disableHidden) return null;
          return (
            <div key={column.accessor}>
              <Checkbox
                name={column.name}
                value={column.isChecked}
                onChange={() => updateColumnItems(column.accessor)}
              >
                <span>{column.name}</span>
              </Checkbox>
            </div>
          );
        })}
      </div>
      <div className={"button-actions"}>
        <Button onClick={onCancelColumns} variant={"inverse"}>
          Cancel
        </Button>
        <Button
          className={"save-button"}
          onClick={() => onSaveColumns(getHiddenColumns(columnItems))}
          variant={"default"}
        >
          Save
        </Button>
      </div>
    </div>
  );
};

EditColumnsModal.propTypes = {
  columns: PropTypes.arrayOf(PropTypes.object), // TODO: create proper interface for this
  hiddenColumns: PropTypes.arrayOf(PropTypes.string),
  onSaveColumns: PropTypes.func,
  onCancelColumns: PropTypes.func,
};

export default EditColumnsModal;
