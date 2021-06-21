import React from "react";
import ReactTooltip from "react-tooltip";

interface IIconToolTipProps {
  text: string;
  isHtml?: boolean;
}

// TODO: handle html text better. possibly use 'children' prop for html
const IssueIconToolTip = (props: IIconToolTipProps): JSX.Element => {
  const { text, isHtml } = props;
  return (
    <div className="icon-tooltip">
      <span data-tip={text} data-html={isHtml}>
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 16 16"
          fill="none"
        >
          <path
            d="M0 8C0 12.4183 3.5817 16 8 16C12.4183 16 16 12.4183 16 8C16 3.5817 12.4183 0 8 0C3.5817 0 0 3.5817 0 8ZM14 8C14 11.3137 11.3137 14 8 14C4.6863 14 2 11.3137 2 8C2 4.6863 4.6863 2 8 2C11.3137 2 14 4.6863 14 8ZM7 12V10H9V12H7ZM7 4V9H9V4H7Z"
            fill="#8B8FA2"
          />
        </svg>
      </span>
      {/* same colour as $core-fleet-blue */}
      <ReactTooltip
        effect={"solid"}
        data-html={isHtml}
        backgroundColor={"#3e4771"}
      />
    </div>
  );
};

export default IssueIconToolTip;
