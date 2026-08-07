import React from "react";
import classnames from "classnames";

const baseClass = "data-set";

interface IDataSetProps {
  title: React.ReactNode;
  value: React.ReactNode;
  orientation?: "horizontal" | "vertical";
  /** When true, aligns the value row by text baseline instead of vertical
   * center. Use this when the value contains only text (with or without
   * tooltips) so neighboring DataSets in a horizontal row share the same
   * baseline. Do NOT use when the value contains icons, buttons, or status
   * indicators that need vertical centering. */
  textOnly?: boolean;
  /** When true, the value wraps onto multiple lines instead of truncating
   * with an ellipsis. Use for free-form prose like "Resolve" or
   * "Description" where the full text needs to be readable. Default behavior
   * (single-line truncation with ellipsis) assumes the consumer wraps the
   * value in a tooltip-on-truncate helper. */
  multiline?: boolean;
  className?: string;
}

const DataSet = ({
  title,
  value,
  orientation = "vertical",
  textOnly = false,
  multiline = false,
  className,
}: IDataSetProps) => {
  const classNames = classnames(baseClass, className, {
    [`${baseClass}__horizontal`]: orientation === "horizontal",
    [`${baseClass}--text-only`]: textOnly,
    [`${baseClass}--multiline`]: multiline,
  });

  return (
    <div className={classNames}>
      <dt>
        {title}
        {orientation === "horizontal" && ":"}
      </dt>
      <dd>{value}</dd>
    </div>
  );
};

export default DataSet;
