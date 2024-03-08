import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IPolicyProps {
  size?: IconSizes;
  color?: Colors;
}
const Policy = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: IPolicyProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      viewBox="0 0 16 16"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fill={COLORS[color]}
        fillRule="evenodd"
        d="M8.314.058a1 1 0 00-.631 0L1.684 2.055l-.586.196-.088.612c-.265 1.855-.27 4.716.605 7.313.886 2.628 2.729 5.1 6.188 5.791l.195.04.196-.04c3.46-.69 5.302-3.163 6.188-5.79.876-2.598.87-5.46.605-7.314l-.088-.612-.586-.196L8.314.058zM2.92 3.752l5.08-1.69 5.08 1.69c.151 1.618.07 3.822-.592 5.785-.72 2.133-2.072 3.87-4.489 4.427-2.416-.558-3.769-2.294-4.488-4.427-.662-1.963-.743-4.167-.591-5.785zm8.151 2.735a.75.75 0 10-1.14-.974L7.459 8.408 6.033 6.97a.75.75 0 00-1.066 1.056l2 2.017a.75.75 0 001.103-.041l3-3.515z"
        clipRule="evenodd"
      />
    </svg>
  );
};

export default Policy;
