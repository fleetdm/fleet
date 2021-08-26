import React, { Component } from "react";
import PropTypes from "prop-types";

import FleetIcon from "components/icons/FleetIcon";
import softwareInterface from "interfaces/software";

const baseClass = "software-vulnerabilities";

class SoftwareVulnerabilities extends Component {
  static propTypes = {
    softwareList: PropTypes.arrayOf(softwareInterface),
  };

  render() {
    const { softwareList } = this.props;

    const vulsList = [];

    const vulnerabilitiesListMaker = () => {
      softwareList.forEach((software) => {
        if (software.vulnerabilities) {
          software.vulnerabilities.forEach((vulnerability) => {
            vulsList.push({
              name: software.name,
              cve: vulnerability.cve,
              details_link: vulnerability.details_link,
            });
          });
        }
      });
    };

    vulnerabilitiesListMaker();

    const renderVulsCount = (list) => {
      if (list.length === 1) {
        return "1 vulnerability detected";
      }
      return `${list.length} vulnerabilities detected`;
    };

    const renderVul = (vul, index) => {
      return (
        <li key={index}>
          Read more about{" "}
          <a href={vul.details_link} target="_blank" rel="noopener noreferrer">
            <em>{vul.name}</em> {vul.cve} vulnerability &nbsp;
            <FleetIcon name="external-link" />
          </a>
        </li>
      );
    };

    // No software vulnerabilities
    if (vulsList.length === 0) {
      return null;
    }

    // Software vulnerabilities
    return (
      <div className={`${baseClass}`}>
        <div className={`${baseClass}__count`}>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 16 16"
            fill="none"
          >
            <path
              d="M0 8C0 12.4183 3.5817 16 8 16C12.4183 16 16 12.4183 16 8C16 3.5817 12.4183 0 8 0C3.5817 0 0 3.5817 0 8ZM14 8C14 11.3137 11.3137 14 8 14C4.6863 14 2 11.3137 2 8C2 4.6863 4.6863 2 8 2C11.3137 2 14 4.6863 14 8ZM7 12V10H9V12H7ZM7 4V9H9V4H7Z"
              fill="#8B8FA2"
            />
          </svg>
          &nbsp;
          {renderVulsCount(vulsList)}
        </div>
        <div className={`${baseClass}__list`}>
          <ul>{vulsList.map((vul, index) => renderVul(vul, index))}</ul>
        </div>
      </div>
    );
  }
}

export default SoftwareVulnerabilities;
