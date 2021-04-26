import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import moment from "moment";

import softwareInterface from "interfaces/software";

const baseClass = "software-list-row";

class SoftwareListRow extends Component {
  static propTypes = {
    software: softwareInterface.isRequired,
  };

  render() {
    const { software } = this.props;
    const { name, source, version } = software;

    return (
      <tr>
        <td className={`${baseClass}__name`}>{name}</td>
        <td className={`${baseClass}__type`}>{source}</td>
        <td className={`${baseClass}__installed-version`}>{version}</td>
      </tr>
    );
  }
}

export default SoftwareListRow;
