import React from "react";
import classnames from "classnames";
import Icon from "components/Icon/Icon";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { secondsToDhms } from "utilities/helpers";

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
      case 3600:
        return "Hourly";
      case 86400:
        return "Daily";
      case 604800:
        return "Weekly";
      default:
        return secondsToDhms(frequency);
    }
  };

  const frequencyIcon = () => {
    if (frequency === 0) {
      return checked ? (
        <Icon size="medium" name="warning" />
      ) : (
        <Icon size="medium" name="clock" color="ui-fleet-black-33" />
      );
    }
    return <Icon size="medium" name="clock" />;
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
