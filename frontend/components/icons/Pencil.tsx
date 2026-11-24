import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IPencil {
  color?: Colors;
  size?: IconSizes;
}

const Pencil = ({ color = "ui-fleet-black-75", size = "medium" }: IPencil) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M9.792 1.956C10.392 1.356 11.32 1.284 12 1.738C12.062 1.772 12.122 1.811 12.175 1.864L12.268 1.956C12.27 1.959 12.273 1.963 12.275 1.965L14.043 3.731L14.163 3.865C14.723 4.552 14.683 5.566 14.044 6.207L12.891 7.359C12.887 7.363 12.884 7.367 12.881 7.371C12.877 7.374 12.873 7.377 12.869 7.38L5.471 14.78C5.33 14.92 5.139 14.999 4.94 15H1.75C1.336 15 1 14.664 1 14.25V11.059C1 10.86 1.079 10.669 1.22 10.529L9.792 1.956ZM2.5 11.37V13.5H4.63L11.289 6.839L9.159 4.71L2.5 11.37ZM11.207 3.017C11.121 2.932 10.989 2.921 10.892 2.985L10.852 3.017L10.22 3.649L12.35 5.779L12.982 5.146C13.079 5.048 13.079 4.889 12.981 4.792L11.207 3.017Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Pencil;
