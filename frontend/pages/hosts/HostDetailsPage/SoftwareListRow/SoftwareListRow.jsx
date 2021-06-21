import React, { Component } from "react";
import IssueIconToolTip from "components/IconToolTip/IssueIconTooltip";

import softwareInterface from "interfaces/software";

const baseClass = "software-list-row";

class SoftwareListRow extends Component {
  static propTypes = {
    software: softwareInterface.isRequired,
  };

  render() {
    const { software } = this.props;
    const { name, source, version, vulnerabilities } = software;

    const TYPE_CONVERSION = {
      apt_sources: "Package (APT)",
      deb_packages: "Package (deb)",
      portage_packages: "Package (Portage)",
      rpm_packages: "Package (RPM)",
      yum_sources: "Package (YUM)",
      npm_packages: "Package (NPM)",
      atom_packages: "Package (Atom)",
      python_packages: "Package (Python)",
      apps: "Application (macOS)",
      chrome_extensions: "Browser plugin (Chrome)",
      firefox_addons: "Browser plugin (Firefox)",
      safari_extensions: "Browser plugin (Safari)",
      homebrew_packages: "Package (Homebrew)",
      programs: "Program (Windows)",
      ie_extensions: "Browser plugin (IE)",
      chocolatey_packages: "Package (Chocolatey)",
      pkg_packages: "Package (pkg)",
    };

    const type = TYPE_CONVERSION[source] || "Unknown";

    const vulnerabilitiesIcon = (vulnerabilities) => {
      if (vulnerabilities && vulnerabilities.length === 1) {
        return <IssueIconToolTip text={"1 vulnerability detected"} />;
      } else if (vulnerabilities) {
        return (
          <IssueIconToolTip
            text={`${vulnerabilities.length} vulnerability detected`}
          />
        );
      }
    };

    return (
      <tr>
        <td className={`${baseClass}__name`}>{vulnerabilitiesIcon()}</td>
        <td className={`${baseClass}__name`}>{name}</td>
        <td className={`${baseClass}__type`}>{type}</td>
        <td className={`${baseClass}__installed-version`}>{version}</td>
      </tr>
    );
  }
}

export default SoftwareListRow;
