import React from "react";
import classnames from "classnames";

import { IOsqueryTable } from "interfaces/osquery_table";
import { osqueryTableNames } from "utilities/osquery_tables";
import { PLATFORM_DISPLAY_NAMES } from "utilities/constants";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import TooltipWrapper from "components/TooltipWrapper";

// @ts-ignore
import AppleIcon from "../../../../assets/images/icon-apple-dark-20x20@2x.png";
import LinuxIcon from "../../../../assets/images/icon-linux-dark-20x20@2x.png";
import WindowsIcon from "../../../../assets/images/icon-windows-dark-20x20@2x.png";
import CloseIcon from "../../../../assets/images/icon-close-black-50-8x8@2x.png";

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
  const displayTypeForDataType = (dataType: string) => {
    switch (dataType) {
      case "TEXT_TYPE":
        return "text";
      case "BIGINT_TYPE":
        return "big int";
      case "INTEGER_TYPE":
        return "integer";
      default:
        return dataType;
    }
  };

  const onSelectTable = (value: string) => {
    onOsqueryTableSelect(value);
  };

  const renderColumns = () => {
    const columns = selectedOsqueryTable?.columns;
    const columnBaseClass = "query-column-list";

    return columns?.map((column) => (
      <li key={column.name} className={`${columnBaseClass}__item`}>
        <span className={`${columnBaseClass}__name`}>
          <TooltipWrapper tipContent={column.description}>
            {column.name}
          </TooltipWrapper>
        </span>
        <div className={`${columnBaseClass}__description`}>
          <span className={`${columnBaseClass}__type`}>
            {displayTypeForDataType(column.type)}
          </span>
        </div>
      </li>
    ));
  };

  const renderTableSelect = () => {
    const tableNames = osqueryTableNames?.map((name: string) => {
      return { label: name, value: name };
    });

    if (!tableNames) {
      return null;
    }

    return (
      <Dropdown
        options={tableNames}
        value={selectedOsqueryTable?.name}
        onChange={onSelectTable}
        placeholder="Choose Table..."
      />
    );
  };

  const { description, platforms } = selectedOsqueryTable || {};
  const iconClasses = classnames([`${baseClass}__icon`], "icon");
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
        <h2 className={`${baseClass}__header`}>Tables</h2>
        {renderTableSelect()}
        <p className={`${baseClass}__description`}>{description}</p>
      </div>
      <div className={`${baseClass}__os-availability`}>
        <h2 className={`${baseClass}__header`}>Compatible with:</h2>
        <ul className={`${baseClass}__platforms`}>
          {platforms?.map((platform) => {
            if (platform === "all") {
              return (
                <li key={platform}>
                  <FleetIcon name="hosts" />{" "}
                  {PLATFORM_DISPLAY_NAMES[platform] || platform}
                </li>
              );
            }
            platform = platform.toLowerCase();
            let icon = (
              <img
                src={AppleIcon}
                alt={`${platform} icon`}
                className={iconClasses}
              />
            );
            if (platform === "linux") {
              icon = (
                <img
                  src={LinuxIcon}
                  alt={`${platform} icon`}
                  className={iconClasses}
                />
              );
            } else if (platform === "windows") {
              icon = (
                <img
                  src={WindowsIcon}
                  alt={`${platform} icon`}
                  className={iconClasses}
                />
              );
            }

            return (
              <li key={platform}>
                {icon} {PLATFORM_DISPLAY_NAMES[platform] || platform}
              </li>
            );
          })}
        </ul>
      </div>
      <div className={`${baseClass}__columns`}>
        <h2 className={`${baseClass}__header`}>Columns</h2>
        <ul className={`${baseClass}__column-list`}>{renderColumns()}</ul>
      </div>
    </>
  );
};

export default QuerySidePanel;
