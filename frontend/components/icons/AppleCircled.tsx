import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IAppleCircledProps {
  size?: IconSizes;
  iconColor?: Colors;
  bgColor?: Colors;
}
const AppleCircled = ({
  size = "extra-large",
  iconColor = "ui-fleet-black-75", // default grey
  bgColor = "ui-blue-10", // default light blue
}: IAppleCircledProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 48 48"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle cx="24" cy="24" r="24" fill={COLORS[bgColor]} />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M27.8083 15.8143C28.711 14.8039 29.3881 13.4146 29.2188 12C27.8647 12.0505 26.2566 12.8083 25.2975 13.8188C24.4229 14.7029 23.6894 16.1427 23.8869 17.5068C25.3539 17.6078 26.8773 16.85 27.8083 15.8143Z"
        fill={COLORS[iconColor]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M31.7296 26.9038H13.2796H31.7296V26.8785C31.3065 26.1207 31.109 24.9335 31.2783 23.9736V23.9483C31.4475 22.6853 32.3503 21.397 33.0838 20.917C33.0838 20.917 33.112 20.917 33.112 20.8918C33.2812 20.7402 33.7326 20.4118 34.0429 20.2097C32.8863 18.6941 31.0244 17.8858 29.106 17.8605C27.1877 17.709 25.1847 18.9467 24.1126 18.9215C23.0688 18.9467 21.489 17.8353 19.8246 17.911H19.7964C19.7681 17.911 19.4578 17.9363 19.4578 17.9363C17.2574 17.9868 15.3108 19.1488 14.1823 20.917C13.731 21.5486 13.2232 23.0389 13.1385 23.9736V24.0241C13.0257 24.6303 13.1385 26.1965 13.2514 26.9543C13.2514 26.9796 13.2514 26.9796 13.2514 27.0048C13.336 27.7626 13.7874 29.2277 14.0977 29.9603C14.408 30.7434 15.2826 32.2337 15.8186 32.9915C16.947 34.4314 18.273 36.0733 20.022 35.9975C21.7147 35.947 22.3353 34.9871 24.3665 35.0124C26.3977 34.9871 26.962 35.947 28.7675 35.9723C30.6012 35.8965 31.7296 34.4566 32.8581 32.9915C33.4223 32.2337 34.2686 30.7939 34.579 29.935C34.6072 29.834 34.6636 29.7077 34.6918 29.6066C33.5634 29.253 32.2092 27.8384 31.7296 26.9038Z"
        fill={COLORS[iconColor]}
      />
    </svg>
  );
};

export default AppleCircled;
