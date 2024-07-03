import React from "react";
import classnames from "classnames";

import {
  ColumnType,
  IQueryTableColumn,
  TableSchemaPlatform,
} from "interfaces/osquery_table";
import TooltipWrapper from "components/TooltipWrapper";
import { buildQueryStringFromParams } from "utilities/url";

interface IColumnListItemProps {
  column: IQueryTableColumn;
  selectedTableName: string;
}

const baseClass = "column-list-item";

const FOOTNOTES = {
  required: "Required in WHERE clause.",
  requires_user_context: "Defaults to root.",
  platform: "Only available on",
};

/**
 * This function is to create the html string for the tooltip. We do this as the
 * current tooltip only supports strings. we can change this when it support ReactNodes
 * in the future.
 */
const renderTooltip = (
  column: IQueryTableColumn,
  selectedTableName: string
) => {
  const renderUserContextFootnote = () => {
    const queryString = buildQueryStringFromParams({
      utm_source: "fleet-ui",
      utm_table: `table-${selectedTableName}`,
    });

    const href = `https://fleetdm.com/guides/osquery-consider-joining-against-the-users-table?${queryString}`;
    const classNames = classnames(
      `${baseClass}__footnote`,
      `${baseClass}__footnote-link`
    );

    return (
      <a href={href} target="__blank" className={classNames}>
        ${FOOTNOTES.requires_user_context}
      </a>
    );
  };

  const renderHiddenFootnote = () => {
    return (
      <span className={`${baseClass}__footnote`}>
        Not returned in SELECT * FROM {selectedTableName}
      </span>
    );
  };

  const renderPlatformFootnotes = (columnPlatforms: TableSchemaPlatform[]) => {
    let platformsCopy;
    switch (columnPlatforms.length) {
      case 1:
        platformsCopy = columnPlatforms[0];
        break;
      case 2:
        platformsCopy = `${columnPlatforms[0]} and ${columnPlatforms[1]}`;
        break;
      case 3:
        platformsCopy = `${columnPlatforms[0]}, ${columnPlatforms[1]}, and ${columnPlatforms[2]}`;
        break;
      default:
        platformsCopy = columnPlatforms.join(", ");
    }
    return (
      <span className={`${baseClass}__footnote`}>
        {FOOTNOTES.platform} {platformsCopy}
      </span>
    );
  };

  return (
    <>
      <span className={`${baseClass}__column-description`}>
        {column.description}
      </span>
      {column.required && (
        <span className={`${baseClass}__footnote`}>{FOOTNOTES.required}</span>
      )}
      {column.requires_user_context && renderUserContextFootnote()}
      {column.hidden && renderHiddenFootnote()}
      {column.platforms && renderPlatformFootnotes(column.platforms)}
    </>
  );
};

const hasFootnotes = (column: IQueryTableColumn) => {
  return (
    column.required ||
    column.requires_user_context ||
    (column.platforms !== undefined && column.platforms.length !== 0)
  );
};

const createTypeDisplayText = (type: ColumnType) => {
  return type.replace("_", " ").toUpperCase();
};

const createListItemClassnames = (column: IQueryTableColumn) => {
  return classnames(`${baseClass}__name`, {
    [`${baseClass}__has-footnotes`]: hasFootnotes(column),
  });
};

const ColumnListItem = ({
  column,
  selectedTableName,
}: IColumnListItemProps) => {
  const columnNameClasses = createListItemClassnames(column);

  return (
    <li key={column.name} className={baseClass}>
      <div className={`${baseClass}__name-wrapper`}>
        <span className={columnNameClasses}>
          <TooltipWrapper
            tipContent={renderTooltip(column, selectedTableName)}
            className={`${baseClass}__tooltip`}
          >
            {column.name}
          </TooltipWrapper>
        </span>
        {column.required && <span className={`${baseClass}__asterisk`}>*</span>}
      </div>
      <span className={`${baseClass}__type`}>
        {createTypeDisplayText(column.type)}
      </span>
    </li>
  );
};

export default ColumnListItem;
