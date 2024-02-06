import React from "react";

const baseClass = "data-set";

interface IDataSetProps {
  title: string;
  value: React.ReactNode;
}

const DataSet = ({ title, value }: IDataSetProps) => {
  return (
    <div className={`${baseClass}`}>
      <dt>{title}</dt>
      <dd>{value}</dd>
    </div>
  );
};

export default DataSet;
