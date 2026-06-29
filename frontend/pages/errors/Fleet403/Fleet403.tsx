import React from "react";
import { Link } from "react-router";

import PATHS from "router/paths";

import { isDarkMode } from "utilities/theme";

// @ts-ignore
import fleetLogoText from "../../../../assets/images/fleet-logo-text-white.svg";
// @ts-ignore
import backgroundImg from "../../../../assets/images/403.svg";
// @ts-ignore
import backgroundImgDark from "../../../../assets/images/403-dark.svg";

const baseClass = "fleet-403";

const Fleet403 = () => (
  <div className={baseClass}>
    <header className="primary-header">
      <Link to={PATHS.DASHBOARD}>
        <img
          className="primary-header__logo"
          src={fleetLogoText}
          alt="Fleet logo"
        />
      </Link>
    </header>
    <img
      src={isDarkMode() ? backgroundImgDark : backgroundImg}
      alt="403 background"
      className="background-image"
    />
    <main>
      <h1>
        <span>Access denied.</span>
      </h1>
      <p>You do not have permissions to access that page.</p>
    </main>
  </div>
);

export default Fleet403;
