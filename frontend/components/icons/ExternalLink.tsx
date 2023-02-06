import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IExternalLinkProps {
  size: IconSizes;
  color: Colors;
}

const ExternalLink = ({
  size = "small",
  color = "core-fleet-blue",
}: IExternalLinkProps) => {
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
        d="M7.33 1.993c.593 0 1.073.473 1.073 1.057 0 .57-.457 1.035-1.029 1.057h-5.23v9.778h9.905v-5.12c0-.57.457-1.035 1.029-1.058h.043c.577 0 1.048.45 1.07 1.015l.001.042v6.178c0 .57-.456 1.035-1.028 1.057L13.12 16H1.07c-.577 0-1.048-.45-1.07-1.015L0 3.05c0-.57.457-1.034 1.029-1.056l.043-.001H7.33ZM14.929 0c.578 0 1.048.45 1.071 1.015L16 5.523c0 .584-.48 1.058-1.072 1.058-.577 0-1.048-.45-1.07-1.015l-.001-.043-.001-1.777-5.854 5.848a1.082 1.082 0 0 1-1.515.009 1.048 1.048 0 0 1-.043-1.46l.033-.036 6-5.992h-2.105c-.577 0-1.048-.45-1.07-1.015L9.3 1.058C9.3.488 9.757.023 10.329 0L14.93 0Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default ExternalLink;
