import React, { Component } from "react";
import { Link } from "react-router";

import PATHS from "router/paths";

import fleetLogoText from "../../../assets/images/fleet-logo-text-white.svg";
import backgroundImg from "../../../assets/images/403.svg";

const baseClass = "fleet-403";

class Fleet403 extends Component {
  render() {
    return (
      <div className={baseClass}>
        <header className="primary-header">
          <Link to={PATHS.HOME}>
            <img
              className="primary-header__logo"
              src={fleetLogoText}
              alt="Fleet logo"
            />
          </Link>
        </header>
        <img
          src={backgroundImg}
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
  }
}

export default Fleet403;
