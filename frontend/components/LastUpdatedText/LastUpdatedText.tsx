import React from "react";
import { formatDistanceToNowStrict } from "date-fns";
import { abbreviateTimeUnits } from "utilities/helpers";

import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "component__last-updated-text";

interface ILastUpdatedTextBase {
  lastUpdatedAt?: string;
}

interface ILastUpdatedTextWithCustomTooltip extends ILastUpdatedTextBase {
  customTooltipText: React.ReactNode;
  whatToRetrieve?: never;
}

interface ILastUpdatedTextWithWhatToRetrieve extends ILastUpdatedTextBase {
  customTooltipText?: never;
  whatToRetrieve: string;
}

const LastUpdatedText = ({
  lastUpdatedAt,
  whatToRetrieve,
  customTooltipText,
}:
  | ILastUpdatedTextWithCustomTooltip
  | ILastUpdatedTextWithWhatToRetrieve): JSX.Element => {
  if (!lastUpdatedAt || lastUpdatedAt === "0001-01-01T00:00:00Z") {
    lastUpdatedAt = "never";
  } else {
    lastUpdatedAt = abbreviateTimeUnits(
      formatDistanceToNowStrict(new Date(lastUpdatedAt), {
        addSuffix: true,
      })
    );
  }

  const tooltipContent = customTooltipText || (
    <>
      Fleet periodically queries all hosts <br />
      to retrieve {whatToRetrieve}.
    </>
  );

  return (
    <span className={baseClass}>
      <TooltipWrapper tipContent={tooltipContent}>
        {`Updated ${lastUpdatedAt}`}
      </TooltipWrapper>
    </span>
  );
};

export default LastUpdatedText;
