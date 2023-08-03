import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IChromeCircledProps {
  size: IconSizes;
  iconColor?: Colors;
  bgColor?: Colors;
}

const ChromeCircled = ({
  size = "extra-large",
  iconColor = "ui-fleet-black-75", // default grey
  bgColor = "ui-blue-10", // default light blue
}: IChromeCircledProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 48 48"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle cx="24" cy="24" r="24" fill={COLORS[bgColor]} />
      <g clipPath="url(#clip0_15434_198874)">
        <path
          d="M12 24C12 21.8156 12.5845 19.7625 13.6064 17.9578L18.7547 26.9203C19.7813 28.7578 21.7453 30 24 30C24.6703 30 25.2703 29.8922 25.9125 29.6906L22.3359 35.8875C16.4953 35.0766 12 30.0609 12 24ZM29.1141 27.075C29.6906 26.175 30 25.0828 30 24C30 22.2094 29.2125 20.6016 27.9703 19.5H35.1281C35.6906 20.8875 36 22.4109 36 24C36 30.6281 30.6281 35.9578 24 36L29.1141 27.075ZM34.3969 18H24C21.0516 18 18.6703 20.0672 18.1172 22.8141L14.5402 16.6158C16.7344 13.8061 20.1562 12 24 12C28.4438 12 32.3203 14.4131 34.3969 18ZM19.875 24C19.875 21.7219 21.7219 19.875 24 19.875C26.2781 19.875 28.125 21.7219 28.125 24C28.125 26.2781 26.2781 28.125 24 28.125C21.7219 28.125 19.875 26.2781 19.875 24Z"
          fill={COLORS[iconColor]}
        />
      </g>
    </svg>
  );
};

export default ChromeCircled;
