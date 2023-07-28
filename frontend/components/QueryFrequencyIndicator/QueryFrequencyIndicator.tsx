import React from "react";
import classnames from "classnames";
import Icon from "components/Icon/Icon";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface IStatusIndicatorProps {
  frequency: number;
  checked: boolean;
}

const generateClassTag = (rawValue: string): string => {
  if (rawValue === DEFAULT_EMPTY_CELL_VALUE) {
    return "indeterminate";
  }
  return rawValue.replace(" ", "-").toLowerCase();
};

const QueryFrequencyIndicator = ({
  frequency,
  checked,
}: IStatusIndicatorProps): JSX.Element => {
  const classTag = generateClassTag(frequency.toString());
  const frequencyClassName = classnames(
    "query-frequency-indicator",
    `query-frequency-indicator--${classTag}`,
    `frequency--${classTag}`
  );
  const readableQueryFrequency = () => {
    switch (frequency) {
      case 0:
        return "Never";
      case 300:
      case 600:
      case 900:
      case 1800: // 5, 10, 15, 30 minutes
        return `${(frequency / 60).toString()} minutes`;
      case 3600:
        return "Hourly";
      case 21600:
      case 43200: // 6, 12 hours
        return `${(frequency / 3600).toString()} hours`;
      case 86400:
        return "Daily";
      case 604800:
        return "Weekly";
      default:
        return "Unknown";
    }
  };

  const frequencyIcon = () => {
    if (frequency === 0) {
      return checked ? (
        <Icon size="small" name="warning" />
      ) : (
        <Icon size="small" name="clock" color="ui-fleet-black-33" />
      );
    }
    return <Icon size="small" name="clock" />;
  };

  return (
    <div
      className={`${frequencyClassName}
        ${frequency === 0 && !checked && "grey"}`}
    >
      {frequencyIcon()}
      {readableQueryFrequency()}
    </div>
  );
};

export default QueryFrequencyIndicator;
