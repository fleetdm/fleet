import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ICentosProps {
  size?: IconSizes;
  color?: Colors;
}

const Centos = ({
  size = "medium",
  color = "ui-fleet-black-75",
}: ICentosProps) => {
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
        d="M2.301 2.36h2.653L3.817 3.524 7.31 7.024v.271h-.298L3.52 3.797 2.3 5.017V2.359ZM11.1 2.36h2.572v2.576L12.508 3.77 8.96 7.295h-.27v-.271l3.546-3.553L11.1 2.36ZM8.69 8.678h.271l3.52 3.525 1.19-1.22v2.658H11.02l1.164-1.166L8.69 8.976v-.298ZM7.012 8.678h.298v.298l-3.547 3.526 1.11 1.139H2.301v-2.577l1.191 1.166 3.52-3.552Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="m2.301 5.722 1.219-1.22 2.788 2.793H2.301V5.722ZM6.308 8.678l-2.816 2.847-1.19-1.166V8.678h4.006Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M0 8.027 1.814 6.21v1.573h5.008l.19.217-.19.19H1.814v1.654L0 8.027ZM9.665 7.295l2.843-2.847 1.164 1.193v1.654H9.665ZM13.672 10.278l-1.191 1.22-2.816-2.82h4.007v1.6Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="m8.988 8 .19-.217h5.008V6.13L16 7.946l-1.814 1.817V8.19H9.178L8.988 8ZM8.69 9.681l2.789 2.794-1.137 1.166H8.69V9.68Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="m7.797 9.166.19-.19.216.19v4.963h1.624L7.96 16l-1.868-1.871h1.706V9.166Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="m4.467 12.502 2.843-2.82v3.959H5.577l-1.11-1.14ZM8.69 6.319l2.789-2.794-1.137-1.139H8.69V6.32Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="m7.797 6.834.19.19.216-.19V1.87h1.624L7.96 0 6.091 1.871h1.706v4.963Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="m4.467 3.498 2.843 2.82V2.387H5.577l-1.11 1.112Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Centos;
