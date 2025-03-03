import React from "react";
import classnames from "classnames";

const baseClass = "data-set";

interface IDataSetProps {
  title: React.ReactNode;
  value: React.ReactNode;
  orientation?: "horizontal" | "vertical";
  className?: string;
}

const DataSet = ({
  title,
  value,
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
      <dd>{value}</dd>
    </div>
  );
};

export default DataSet;
