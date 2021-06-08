import React, { Component } from "react";

import configInterface from "interfaces/config";
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import userInterface from "interfaces/user";

class SiteNavHeader extends Component {
  static propTypes = {
    config: configInterface,
    user: userInterface,
  };

  render() {
    const {
      config: { org_logo_url: orgLogoURL },
      user,
    } = this.props;

    const { username } = user;

    const headerBaseClass = "site-nav-header";

    return (
      <header className={headerBaseClass}>
        <div className={`${headerBaseClass}__inner`}>
          <OrgLogoIcon
            className={`${headerBaseClass}__logo`}
            src={orgLogoURL}
          />
        </div>
      </header>
    );
  }
}

export default SiteNavHeader;
