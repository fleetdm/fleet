import React from "react";
import classnames from "classnames";

import { ColumnType, IQueryTableColumn } from "interfaces/osquery_table";
import { PLATFORM_DISPLAY_NAMES } from "utilities/constants";
import TooltipWrapper from "components/TooltipWrapper";

interface IColumnListItemProps {
  column: IQueryTableColumn;
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
const createTooltipHtml = (column: IQueryTableColumn) => {
  const toolTipHtml = [];

  const descriptionHtml = `<span class="${baseClass}__column-description">${column.description}</span>`;
  toolTipHtml.push(descriptionHtml);

  if (column.required) {
    toolTipHtml.push(
      `<span class="${baseClass}__footnote">${FOOTNOTES.required}</span>`
    );
  }

  if (column.requires_user_context) {
    toolTipHtml.push(
      `<span class="${baseClass}__footnote">${FOOTNOTES.requires_user_context}</span>`
    );
  }

  if (column.platforms?.length === 1) {
    const platform = column.platforms[0];
    toolTipHtml.push(
      `<span class="${baseClass}__footnote">${FOOTNOTES.platform} ${platform}</span>`
    );
  }

  if (column.platforms?.length === 2) {
    const platform1 = PLATFORM_DISPLAY_NAMES[column.platforms[0]];
    const platform2 = PLATFORM_DISPLAY_NAMES[column.platforms[1]];
    toolTipHtml.push(
      `<span class="${baseClass}__footnote">${FOOTNOTES.platform} ${platform1} and ${platform2}.</span>`
    );
  }

  const tooltip = toolTipHtml.join("");
  return tooltip;
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

const ColumnListItem = ({ column }: IColumnListItemProps) => {
  const columnNameClasses = createListItemClassnames(column);

  return (
    <li key={column.name} className={baseClass}>
      <div className={`${baseClass}__name-wrapper`}>
        <span className={columnNameClasses}>
          <TooltipWrapper
            tipContent={createTooltipHtml(column)}
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
