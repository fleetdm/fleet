import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IiOSProps {
  size: IconSizes;
  color?: Colors;
}

const iOS = ({ size = "medium", color = "ui-fleet-black-75" }: IiOSProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
    >
      <rect
        x="5.25"
        y=".75"
        width="13.5"
        height="22.5"
        rx="1.25"
        stroke={COLORS[color]}
        strokeWidth="1.5"
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M13.665 7.875c.053.442-.159.876-.441 1.192-.29.324-.767.56-1.225.529-.062-.426.167-.876.44-1.153.3-.315.803-.552 1.226-.568Zm.784 4.65v.007c.15.293.573.735.926.845a.7.7 0 0 1-.018.051c-.006.018-.013.036-.017.052-.097.268-.362.718-.538.955-.353.458-.705.908-1.278.931-.27-.003-.45-.074-.634-.145-.203-.079-.41-.159-.742-.155-.34-.004-.554.08-.76.162-.179.07-.352.139-.597.146-.547.024-.961-.49-1.314-.94-.167-.236-.44-.702-.538-.946-.097-.23-.238-.687-.264-.924v-.016c-.035-.237-.07-.726-.035-.915v-.016c.026-.292.185-.758.326-.955.352-.553.96-.916 1.648-.932l.106-.008h.009c.294-.013.579.092.83.184.194.071.368.135.51.132.137.003.324-.06.538-.133.306-.104.669-.227 1.022-.199.6.008 1.182.26 1.543.735a3.243 3.243 0 0 0-.29.213c0 .008-.01.008-.01.008-.229.15-.51.552-.564.947v.008c-.053.3.01.67.141.907Z"
        fill={COLORS[color]}
      />
      <path
        d="M9.75 1h4.5v1a.5.5 0 0 1-.5.5h-3.5a.5.5 0 0 1-.5-.5V1Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default iOS;
