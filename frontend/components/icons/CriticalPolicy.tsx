import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ICriticalPolicyProps {
  color?: Colors;
  size?: IconSizes;
}

const CriticalPolicy = ({
  color = "core-fleet-blue",
  size = "small",
}: ICriticalPolicyProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g clipPath="url(#clip0_210_11264)">
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M8.31628 0.0513167C8.11101 -0.0171056 7.88909 -0.0171056 7.68382 0.0513167L1.68382 2.05132L1.09751 2.24676L1.0101 2.85858C0.744777 4.71586 0.739144 7.58148 1.61486 10.1817C2.50082 12.8123 4.3438 15.2886 7.80394 15.9806L8.00005 16.0198L8.19617 15.9806C11.6563 15.2886 13.4993 12.8123 14.3852 10.1817C15.261 7.58148 15.2553 4.71586 14.99 2.85858L14.9026 2.24676L14.3163 2.05132L8.31628 0.0513167ZM3.51025 9.54333C2.84797 7.57686 2.76666 5.36854 2.91876 3.74786L8.00005 2.05409L13.0813 3.74786C13.2334 5.36854 13.1521 7.57686 12.4899 9.54333C11.7701 11.6806 10.4166 13.4188 8.00005 13.9772C5.58348 13.4188 4.23004 11.6806 3.51025 9.54333ZM11.0709 6.48649C11.3396 6.17124 11.3018 5.69787 10.9865 5.42919C10.6713 5.16051 10.1979 5.19826 9.92924 5.51351L7.45871 8.41227L6.03302 6.97231C5.74159 6.67797 5.26672 6.67561 4.97237 6.96704C4.67803 7.25847 4.67566 7.73334 4.9671 8.02769L6.9671 10.0477C7.11479 10.1969 7.31825 10.2773 7.52803 10.2695C7.73781 10.2616 7.9347 10.1663 8.07087 10.0065L11.0709 6.48649Z"
          fill={COLORS[color]}
        />
      </g>
      <defs>
        <clipPath id="clip0_210_11264">
          <rect width="16" height="16" fill="white" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default CriticalPolicy;
