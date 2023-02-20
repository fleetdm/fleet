import React from "react";

import { IOsQueryTable } from "interfaces/osquery_table";
import { osqueryTableNames } from "utilities/osquery_tables";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import FleetMarkdown from "components/FleetMarkdown";

import QueryTableColumns from "./QueryTableColumns";
import QueryTablePlatforms from "./QueryTablePlatforms";

// @ts-ignore
import CloseIcon from "../../../../assets/images/icon-close-black-50-8x8@2x.png";
import QueryTableExample from "./QueryTableExample";
import QueryTableNotes from "./QueryTableNotes";
import EventedTableTag from "./EventedTableTag";
import CustomLink from "components/CustomLink";

interface IQuerySidePanel {
  selectedOsqueryTable: IOsQueryTable;
  onOsqueryTableSelect: (tableName: string) => void;
  onClose?: () => void;
}

const baseClass = "query-side-panel";

const QuerySidePanel = ({
  selectedOsqueryTable,
  onOsqueryTableSelect,
  onClose,
}: IQuerySidePanel): JSX.Element => {
  const {
    name,
    description,
    platforms,
    columns,
    examples,
    notes,
    evented,
  } = selectedOsqueryTable;

  const onSelectTable = (value: string) => {
    onOsqueryTableSelect(value);
  };

  const renderTableSelect = () => {
    const tableNames = osqueryTableNames?.map((tableName: string) => {
      return { label: tableName, value: tableName };
    });

    return (
      <Dropdown
        options={tableNames}
        value={name}
        onChange={onSelectTable}
        placeholder="Choose Table..."
        className={`${baseClass}__table-select`}
      />
    );
  };

  return (
    <>
      <div
        role="button"
        className={`${baseClass}__close-button`}
        tabIndex={0}
        onClick={onClose}
      >
        <img alt="Close sidebar" src={CloseIcon} />
      </div>
      <div className={`${baseClass}__choose-table`}>
        <h2 className={`${baseClass}__header`}>
          Tables
          <span className={`${baseClass}__table-count`}>
            {osqueryTableNames.length}
          </span>
        </h2>
        {renderTableSelect()}
      </div>
      {evented && <EventedTableTag selectedTableName={name} />}
      <div className={`${baseClass}__description`}>
        <FleetMarkdown markdown={description} />
      </div>
      <QueryTablePlatforms platforms={platforms} />
      <QueryTableColumns columns={columns} />
      {examples && <QueryTableExample example={examples} />}
      {notes && <QueryTableNotes notes={notes} />}
      <CustomLink
        url={`https://www.fleetdm.com/tables/${name}`}
        text="Source"
        newTab
      />
    </>
  );
};

export default QuerySidePanel;
