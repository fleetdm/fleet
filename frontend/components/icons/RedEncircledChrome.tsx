import React from "react";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IRedEncircledChrome {
  size: IconSizes;
}

const RedEncircledChrome = ({ size = "extra-large" }: IRedEncircledChrome) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 49 49"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle cx="24.5" cy="24.5" r="24" fill="#FF5C83" />
      <g clipPath="url(#clip0_17779_222153)">
        <path
          d="M12.5 24.5C12.5 22.3156 13.0845 20.2625 14.1064 18.4578L19.2547 27.4203C20.2813 29.2578 22.2453 30.5 24.5 30.5C25.1703 30.5 25.7703 30.3922 26.4125 30.1906L22.8359 36.3875C16.9953 35.5766 12.5 30.5609 12.5 24.5ZM29.6141 27.575C30.1906 26.675 30.5 25.5828 30.5 24.5C30.5 22.7094 29.7125 21.1016 28.4703 20H35.6281C36.1906 21.3875 36.5 22.9109 36.5 24.5C36.5 31.1281 31.1281 36.4578 24.5 36.5L29.6141 27.575ZM34.8969 18.5H24.5C21.5516 18.5 19.1703 20.5672 18.6172 23.3141L15.0402 17.1158C17.2344 14.3061 20.6562 12.5 24.5 12.5C28.9438 12.5 32.8203 14.9131 34.8969 18.5ZM20.375 24.5C20.375 22.2219 22.2219 20.375 24.5 20.375C26.7781 20.375 28.625 22.2219 28.625 24.5C28.625 26.7781 26.7781 28.625 24.5 28.625C22.2219 28.625 20.375 26.7781 20.375 24.5Z"
          fill="white"
        />
      </g>
      <defs>
        <clipPath id="clip0_17779_222153">
          <rect
            width="24"
            height="24"
            fill="white"
            transform="translate(12.5 12.5)"
          />
        </clipPath>
      </defs>
    </svg>
  );
};
export default RedEncircledChrome;
