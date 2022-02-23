import React from "react";
import formatDistanceToNowStrict from "date-fns/formatDistanceToNowStrict";

import TooltipWrapper from "components/TooltipWrapper";

const renderLastUpdatedText = (
  lastUpdatedAt: string,
  whatToRetrieve: string
): JSX.Element => {
  if (!lastUpdatedAt || lastUpdatedAt === "0001-01-01T00:00:00Z") {
    lastUpdatedAt = "never";
  } else {
    lastUpdatedAt = formatDistanceToNowStrict(new Date(lastUpdatedAt), {
      addSuffix: true,
    });
  }

  return (
    <span className="last-updated">
      <TooltipWrapper
        tipContent={`Fleet periodically queries all hosts to retrieve ${whatToRetrieve}`}
      >
        {`Last updated ${lastUpdatedAt}`}
      </TooltipWrapper>
    </span>
  );
};

export default renderLastUpdatedText;
