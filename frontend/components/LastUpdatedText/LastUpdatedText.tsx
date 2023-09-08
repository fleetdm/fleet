import React from "react";
import formatDistanceToNowStrict from "date-fns/formatDistanceToNowStrict";
import { abbreviateTimeUnits } from "utilities/helpers";

import NewTooltipWrapper from "components/NewTooltipWrapper";

const baseClass = "component__last-updated-text";

interface ILastUpdatedTextProps {
  lastUpdatedAt?: string;
  whatToRetrieve: string;
}
const LastUpdatedText = ({
  lastUpdatedAt,
  whatToRetrieve,
}: ILastUpdatedTextProps): JSX.Element => {
  if (!lastUpdatedAt || lastUpdatedAt === "0001-01-01T00:00:00Z") {
    lastUpdatedAt = "never";
  } else {
    lastUpdatedAt = abbreviateTimeUnits(
      formatDistanceToNowStrict(new Date(lastUpdatedAt), {
        addSuffix: true,
      })
    );
  }

  return (
    <span className={baseClass}>
      <NewTooltipWrapper
        tipContent={
          <>
            Fleet periodically queries all hosts <br />
            to retrieve {whatToRetrieve}.
          </>
        }
      >
        {`Updated ${lastUpdatedAt}`}
      </NewTooltipWrapper>
    </span>
  );
};

export default LastUpdatedText;
