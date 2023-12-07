import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IAppleProps {
  size?: IconSizes;
  color?: Colors;
}

const Apple = ({
  size = "medium",
  color = "ui-fleet-black-75",
}: IAppleProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M10.903 2.543c.602-.674 1.053-1.6.94-2.543-.902.034-1.974.539-2.614 1.212-.583.59-1.072 1.55-.94 2.46.978.067 1.994-.439 2.614-1.13ZM13.518 9.936h-12.3 12.3v-.017c-.283-.505-.414-1.297-.301-1.937v-.016c.112-.842.714-1.701 1.203-2.021 0 0 .02 0 .02-.017.112-.101.413-.32.62-.455-.771-1.01-2.013-1.55-3.292-1.566-1.279-.101-2.614.724-3.328.707-.696.017-1.75-.724-2.86-.673h-.018c-.019 0-.226.017-.226.017A4.286 4.286 0 0 0 1.82 5.945c-.3.42-.64 1.414-.696 2.037v.034c-.075.404 0 1.448.076 1.954v.033c.056.505.357 1.482.564 1.97.207.523.79 1.516 1.147 2.021.752.96 1.636 2.055 2.802 2.004 1.129-.033 1.543-.673 2.897-.656 1.354-.017 1.73.623 2.934.64 1.222-.051 1.975-1.01 2.727-1.988.376-.505.94-1.465 1.147-2.037.019-.068.056-.152.075-.22-.752-.235-1.655-1.178-1.975-1.801Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Apple;
