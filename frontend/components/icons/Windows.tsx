import React from "react";

interface IWindowsProps {
  size: "small" | "medium" | "large";
  color?: "coreFleetBlack" | "coreVibrantBlue" | "uiFleetBlack75";
}

const FLEET_COLORS = {
  coreFleetBlack: "#192147",
  coreVibrantBlue: "#6a67fe",
  uiFleetBlack75: "#515774",
};

const WIDTH_SIZE_MAP = {
  small: "12",
  medium: "16",
  large: "24",
};

const HEIGHT_SIZE_MAP = {
  small: "12.5",
  medium: "17",
  large: "25",
};

const Windows = ({
  size = "medium",
  color = "uiFleetBlack75",
}: IWindowsProps) => {
  return (
    <svg
      width={WIDTH_SIZE_MAP[size]}
      height={HEIGHT_SIZE_MAP[size]}
      viewBox="0 0 16 17"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M1.09229 13.6417L6.28392 14.6796V8.81934H1.09229V13.6417Z"
        fill={FLEET_COLORS[color]}
      />
      <path
        d="M1.09229 8.16465H6.28392V2.3363L1.09229 3.37423V8.16465Z"
        fill={FLEET_COLORS[color]}
      />
      <path
        d="M7.1095 8.16467H15.4923V0.5L7.1095 2.17665V8.16467Z"
        fill={FLEET_COLORS[color]}
      />
      <path
        d="M7.1095 14.8393L15.4923 16.5V8.81934H7.1095V14.8393Z"
        fill={FLEET_COLORS[color]}
      />
    </svg>
  );
};

export default Windows;
