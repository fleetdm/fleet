import React from "react";
import classnames from "classnames";

const baseClass = "data-set";

interface IDataSetProps {
  title: React.ReactNode;
  value: React.ReactNode;
  className?: string;
}

const DataSet = ({ title, value, className }: IDataSetProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <div className={classNames}>
      <dt>{title}</dt>
      <dd>{value}</dd>
    </div>
  );
};

export default DataSet;
