import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IChromeOSProps {
  size: IconSizes;
  color?: Colors;
}

const ChromeOS = ({
  size = "medium",
  color = "ui-fleet-black-75",
}: IChromeOSProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M0.5 12.5C0.5 10.3156 1.08453 8.2625 2.10641 6.45781L7.25469 15.4203C8.28125 17.2578 10.2453 18.5 12.5 18.5C13.1703 18.5 13.7703 18.3922 14.4125 18.1906L10.8359 24.3875C4.99531 23.5766 0.5 18.5609 0.5 12.5ZM17.6141 15.575C18.1906 14.675 18.5 13.5828 18.5 12.5C18.5 10.7094 17.7125 9.10156 16.4703 8H23.6281C24.1906 9.3875 24.5 10.9109 24.5 12.5C24.5 19.1281 19.1281 24.4578 12.5 24.5L17.6141 15.575ZM22.8969 6.5H12.5C9.55156 6.5 7.17031 8.56719 6.61719 11.3141L3.04016 5.11578C5.23438 2.30609 8.65625 0.5 12.5 0.5C16.9438 0.5 20.8203 2.91313 22.8969 6.5ZM8.375 12.5C8.375 10.2219 10.2219 8.375 12.5 8.375C14.7781 8.375 16.625 10.2219 16.625 12.5C16.625 14.7781 14.7781 16.625 12.5 16.625C10.2219 16.625 8.375 14.7781 8.375 12.5Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default ChromeOS;
