import React from "react";
import ReactTooltip from "react-tooltip";
import formatDistanceToNowStrict from "date-fns/formatDistanceToNowStrict";
import { kebabCase } from "lodash";

import QuestionIcon from "../../../../../assets/images/icon-question-16x16@2x.png";

const renderLastUpdatedText = (
  lastUpdatedAt: string,
  whatToRetrieve: string
) => {
  if (!lastUpdatedAt || lastUpdatedAt === "0001-01-01T00:00:00Z") {
    lastUpdatedAt = "never";
  } else {
    lastUpdatedAt = formatDistanceToNowStrict(new Date(lastUpdatedAt), {
      addSuffix: true,
    });
  }

  return (
    <span className="last-updated">
      {`Last updated ${lastUpdatedAt}`}
      <span className={`tooltip`}>
        <span
          className={`tooltip__tooltip-icon`}
          data-tip
          data-for={`last-updated-tooltip-${kebabCase(whatToRetrieve)}`}
          data-tip-disable={false}
        >
          <img alt="question icon" src={QuestionIcon} />
        </span>
        <ReactTooltip
          place="top"
          type="dark"
          effect="solid"
          backgroundColor="#3e4771"
          id={`last-updated-tooltip-${kebabCase(whatToRetrieve)}`}
          data-html
        >
          <span className={`tooltip__tooltip-text`}>
            Fleet periodically
            <br />
            queries all hosts
            <br />
            to retrieve {whatToRetrieve}
          </span>
        </ReactTooltip>
      </span>
    </span>
  );
};

export default renderLastUpdatedText;
