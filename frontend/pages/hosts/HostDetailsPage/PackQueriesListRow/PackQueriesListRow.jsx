import React, { Component } from "react";

// THIS IS COMING FROM ZACH
// import softwareInterface from "interfaces/software";

const baseClass = "pack-queries-list-row";

class PackQueriesListRow extends Component {
  // static propTypes = {
  //   software: softwareInterface.isRequired,
  // };

  // BRING IN PROPS BEFORE THIS CAN RENDER
  // static propTypes = {
  //   query: PropTypes.object, // fix this proptype
  // };

  render() {
    const { query } = this.props;
    const { name, description, frequency, last_executed } = query;

    return (
      <tr>
        <td className={`${baseClass}__name`}>{name}</td>
        <td className={`${baseClass}__description`}>{description}</td>
        <td className={`${baseClass}__frequency`}>{frequency}</td>
        <td className={`${baseClass}__last-run`}>{last_executed}</td>
      </tr>
    );
  }
}

export default PackQueriesListRow;
