import React from "react";
import classnames from "classnames";
import TooltipTruncatedText from "components/TooltipTruncatedText";

const baseClass = "data-set";

interface IDataSetProps {
  title: React.ReactNode;
  value: React.ReactNode;
  //* Whether to truncate overflowing value and display it in full in a tooltipo */
  tooltipTruncate?: boolean;
  orientation?: "horizontal" | "vertical";
  className?: string;
}

const DataSet = ({
  title,
  value,
  tooltipTruncate = false,
  orientation = "vertical",
  className,
}: IDataSetProps) => {
  const classNames = classnames(baseClass, className, {
    [`${baseClass}__horizontal`]: orientation === "horizontal",
  });

  return (
    <div className={classNames}>
      <dt>
        {title}
        {orientation === "horizontal" && ":"}
      </dt>
      <dd>
        {tooltipTruncate ? <TooltipTruncatedText value={value} /> : value}
      </dd>
    </div>
  );
};

export default DataSet;
