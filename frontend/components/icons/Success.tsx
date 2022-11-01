import React from "react";

interface ISuccessProps {
  color?: "coreVibrantBlue" | "coreFleetBlack";
}

const FLEET_COLORS = {
  coreFleetBlack: "#192147",
  coreVibrantBlue: "#6a67fe",
};

const Success = ({ color = "coreFleetBlack" }: ISuccessProps) => {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M0 8c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8-8 3.58-8 8Zm11.29-2.71a1.003 1.003 0 0 1 1.42 1.42l-5 5c-.18.18-.43.29-.71.29-.28 0-.53-.11-.71-.29l-3-3a1.003 1.003 0 0 1 1.42-1.42L7 9.59l4.29-4.3Z"
        fill="#515774"
      />
    </svg>
  );
};

export default Success;
