import React, { Component } from "react";
import { Link } from "react-router";
import PATHS from "router/paths";

import configInterface from "interfaces/config";
import OrgLogoIcon from "components/icons/OrgLogoIcon";

class SiteNavHeader extends Component {
  static propTypes = {
    config: configInterface,
  };

  render() {
    const {
      config: { org_logo_url: orgLogoURL },
    } = this.props;

    const headerBaseClass = "site-nav-header";

    return (
      <header className={headerBaseClass}>
        <div className={`${headerBaseClass}__inner`}>
          <Link to={PATHS.HOMEPAGE} className={`${headerBaseClass}__home-icon`}>
            <OrgLogoIcon
              className={`${headerBaseClass}__logo`}
              src={orgLogoURL}
            />
          </Link>
        </div>
      </header>
    );
  }
}

export default SiteNavHeader;
