import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface INavSchedule {
  color?: Colors;
}
const NavSchedule = ({ color = "core-fleet-white" }: INavSchedule) => {
  return (
    <svg
      width="16"
      height="16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <g clipPath="url(#a)" fill={COLORS[color]}>
        <path d="M15.106 8.1v-6L7.553 0 0 2.2v10.9l7.553 2.2.397-.2c.795.5 1.79.9 2.882.9C13.615 16 16 13.7 16 10.9c-.1-1.1-.398-2-.894-2.8Zm-6.957-3 5.764-1.7v3.4a5.114 5.114 0 0 0-3.18-1.1c-.894 0-1.789.3-2.584.8V5.1Zm4.174-2.5L7.553 4l-4.77-1.4 4.77-1.4 4.77 1.4Zm-11.13.8 5.764 1.7v2.4c-.796.9-1.292 2.1-1.292 3.4 0 1.1.298 2 .894 2.9l-5.366-1.5V3.4Zm9.54 11.4c-2.186 0-3.975-1.8-3.975-3.9 0-2.2 1.789-3.9 3.975-3.9s3.975 1.8 3.975 3.9c0 2.1-1.789 3.9-3.975 3.9Z" />
        <path d="M12.3 10.9h-.9V9c0-.3-.3-.6-.6-.6s-.6.2-.6.6v3h2.1c.3 0 .6-.3.6-.6s-.2-.5-.6-.5Z" />
      </g>
      <defs>
        <clipPath id="a">
          <path fill="#fff" d="M0 0h16v16H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default NavSchedule;
