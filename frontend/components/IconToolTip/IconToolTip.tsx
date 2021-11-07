import React from "react";
import ReactTooltip from "react-tooltip";

export interface IIconToolTipProps {
  text: string;
  isHtml?: boolean;
  issue?: boolean;
}

// TODO: handle html text better. possibly use 'children' prop for html
const IconToolTip = ({
  text,
  isHtml,
  issue,
}: IIconToolTipProps): JSX.Element => {
  let svgIcon = (
    <svg
      width="16"
      height="17"
      viewBox="0 0 16 17"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle cx="8" cy="8.59961" r="8" fill="#6A67FE" />
      <path
        d="M7.49605 10.1893V9.70927C7.49605 9.33327 7.56405 8.98527 7.70005 8.66527C7.84405 8.34527 8.08405 7.99727 8.42005 7.62127C8.67605 7.34127 8.85205 7.10127 8.94805 6.90127C9.05205 6.70127 9.10405 6.48927 9.10405 6.26527C9.10405 6.00127 9.00805 5.79327 8.81605 5.64127C8.62405 5.48927 8.35205 5.41326 8.00005 5.41326C7.21605 5.41326 6.49205 5.70127 5.82805 6.27727L5.32405 5.12527C5.66005 4.82127 6.07605 4.57727 6.57205 4.39327C7.07605 4.20927 7.58405 4.11727 8.09605 4.11727C8.60005 4.11727 9.04005 4.20127 9.41605 4.36927C9.80005 4.53727 10.096 4.76927 10.304 5.06527C10.52 5.36127 10.628 5.70927 10.628 6.10927C10.628 6.47727 10.544 6.82127 10.376 7.14127C10.216 7.46127 9.92805 7.80927 9.51205 8.18527C9.13605 8.52927 8.87605 8.82927 8.73205 9.08527C8.58805 9.34127 8.49605 9.59727 8.45605 9.85327L8.40805 10.1893H7.49605ZM7.11205 12.6973V11.0293H8.79205V12.6973H7.11205Z"
        fill="white"
      />
    </svg>
  );

  if (issue) {
    svgIcon = (
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="24"
        height="24"
        viewBox="0 -6.5 24 24"
        fill="none"
      >
        <path
          d="M0 8C0 12.4183 3.5817 16 8 16C12.4183 16 16 12.4183 16 8C16 3.5817 12.4183 0 8 0C3.5817 0 0 3.5817 0 8ZM14 8C14 11.3137 11.3137 14 8 14C4.6863 14 2 11.3137 2 8C2 4.6863 4.6863 2 8 2C11.3137 2 14 4.6863 14 8ZM7 12V10H9V12H7ZM7 4V9H9V4H7Z"
          fill="#8B8FA2"
        />
      </svg>
    );
  }
  return (
    <div className="icon-tooltip">
      <span data-tip={text} data-html={isHtml}>
        {svgIcon}
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

export default IconToolTip;
