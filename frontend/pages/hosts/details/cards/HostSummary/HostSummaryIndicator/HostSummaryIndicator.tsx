import React from "react";

interface IHostSummaryIndicatorProps {
  title: string;
  children: JSX.Element;
}

const HostSummaryIndicator = ({
  title,
  children,
}: IHostSummaryIndicatorProps): JSX.Element => {
  return (
    <div className="info-flex__item info-flex__item--title">
      <span className="info-flex__header">{title}</span>
      {children}
    </div>
  );
};

export default HostSummaryIndicator;
