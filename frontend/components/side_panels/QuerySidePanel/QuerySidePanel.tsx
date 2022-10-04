import React from "react";

import { IOsqueryTable } from "interfaces/osquery_table";
import { IOsqueryPlatform } from "interfaces/platform";
import { osqueryTableNames } from "utilities/osquery_tables";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import FleetMarkdown from "components/FleetMarkdown";
import Icon from "components/Icon";

import QueryTableColumns from "./QueryTableColumns";
import QueryTablePlatforms from "./QueryTablePlatforms";
// TODO: change path after moving type to common location.
import { IQueryTableColumn } from "./QueryTableColumns/QueryTableColumns";

// @ts-ignore
import CloseIcon from "../../../../assets/images/icon-close-black-50-8x8@2x.png";
import QueryTableExample from "./QueryTableExample";
import QueryTableNotes from "./QueryTableNotes";

interface QueryTableData {
  name: string;
  description: string;
  url: string;
  hidden: boolean;
  platforms: IOsqueryPlatform[];
  evented: boolean;
  notes?: string;
  examples?: string;
  columns: IQueryTableColumn[];
}

const MOCK_TABLE_DATA: QueryTableData[] = [
  {
    name: "account_policy_data",
    description:
      'changing this to [something else](https://example.com) and another <a href="https://test.com" target="__blank">link</a>', // supports markdown
    url:
      "https://github.com/osquery/osquery/blob/master/specs/darwin/account_policy_data.table",
    hidden: true,
    platforms: [
      "darwin",
      "windows", // Note that this array was overwritten, not merged. Darwin vs. Windows!
      "linux",
    ],
    evented: false,
    notes: "* on M1 macs, this will return XYZ, ABC\n* on M2 macs, this will", // supports markdown
    // examples: "// This will get everything\nSELECT * FROM account_policy_data;", // supports markdown
    examples:
      "changing this to [something else](https://example.com)\nSELECT username, uid, failed_login_timestamp FROM account_policy_data JOIN users USING(pid) WHERE failed__login_timestamp > 0;", // supports markdown
    columns: [
      {
        // this column is unchanged because the fleet schema did not reference it.
        name: "uid",
        description: "User ID",
        type: "bigint",
        required: true,
      },
      {
        name: "creation_time",
        description: "When the account was first created",
        type: "double",
        required: false,
        platforms: [
          "darwin",
          "linux", // this was added. Note that it is a hard overwrite, not a merge.
        ],
        requires_user_context: true,
      },
      {
        name: "before not required",
        description: "When the account was first created",
        type: "double",
        required: false,
        platforms: [],
        requires_user_context: false,
      },
      {
        name: "before required",
        description: "When the account was first created",
        type: "double",
        required: true,
        platforms: [
          "darwin",
          "windows", // this was added. Note that it is a hard overwrite, not a merge.
        ],
        requires_user_context: false,
      },
    ],
  },
];

interface IQuerySidePanel {
  selectedOsqueryTable: IOsqueryTable;
  onOsqueryTableSelect: (tableName: string) => void;
  onClose?: () => void;
}

const baseClass = "query-side-panel";

const QuerySidePanel = ({
  selectedOsqueryTable,
  onOsqueryTableSelect,
  onClose,
}: IQuerySidePanel): JSX.Element => {
  const onSelectTable = (value: string) => {
    onOsqueryTableSelect(value);
  };

  const renderTableSelect = () => {
    const tableNames = osqueryTableNames?.map((name: string) => {
      return { label: name, value: name };
    });

    return (
      <Dropdown
        options={tableNames}
        value={selectedOsqueryTable?.name}
        onChange={onSelectTable}
        placeholder="Choose Table..."
      />
    );
  };

  const { description } = selectedOsqueryTable || {};

  const MOCK_DATA = MOCK_TABLE_DATA[0];
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
            {MOCK_TABLE_DATA.length}
          </span>
        </h2>
        {renderTableSelect()}
      </div>
      {/* {selectedOsqueryTable.evented && ( */}
      {true && (
        <div className={`${baseClass}__evented-table-tag`}>
          <Icon name="calendar-check" className={`${baseClass}__event-icon`} />
          <span>EVENTED TABLE</span>
        </div>
      )}
      <div className={`${baseClass}__description`}>
        <FleetMarkdown markdown={MOCK_DATA.description} />
      </div>
      <QueryTablePlatforms platforms={MOCK_DATA.platforms} />
      <QueryTableColumns columns={MOCK_DATA.columns} />
      {MOCK_DATA.examples && <QueryTableExample example={MOCK_DATA.examples} />}
      {MOCK_DATA.notes && <QueryTableNotes notes={MOCK_DATA.notes} />}
    </>
  );
};

export default QuerySidePanel;
