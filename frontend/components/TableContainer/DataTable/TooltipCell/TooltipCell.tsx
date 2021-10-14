import React from "react";
import classnames from "classnames";

import ReactTooltip from "react-tooltip";

import QuestionIcon from "../../../../../assets/images/icon-question-16x16@2x.png";

interface ITooltipCellProps {
  value: any;
  customIdPrefix?: string;
}

const TooltipCell = (props: ITooltipCellProps): JSX.Element => {
  const { value, customIdPrefix } = props;

  const tooltipClassName = classnames(
    "data-table__tooltip",
    `data-table__tooltip--${customIdPrefix}`
  );

  return (
    <span>
      {value.name}
      {value.bundle_identifier && (
        <>
          {" "}
          <span
            className={`software-name tooltip__tooltip-icon`}
            data-tip
            data-for={`software-name__${value.id.toString()}`}
            data-tip-disable={false}
          >
            <img alt="bundle identifier" src={QuestionIcon} />
          </span>
          <ReactTooltip
            place="bottom"
            type="dark"
            effect="solid"
            backgroundColor="#3e4771"
            id={`software-name__${value.id.toString()}`}
            data-html
          >
            <span className={`tooltip__tooltip-text`}>
              <b>Bundle identifier: </b>
              <br />
              {value.bundle_identifier}
            </span>
          </ReactTooltip>{" "}
        </>
      )}
    </span>
  );
};
export default TooltipCell;
