import React, { Component } from "react";

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
