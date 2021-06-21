import React, { Component } from "react";

// import softwareInterface from "interfaces/software";

const baseClass = "hosts-user-list-row";

class HostUsersListRow extends Component {
  // static propTypes = {
  //   software: softwareInterface.isRequired,
  // };

  render() {
    const { hostUser } = this.props;
    const { username, groupname } = hostUser;

    return (
      <tr>
        <td className={`${baseClass}__username`}>{username}</td>
        <td className={`${baseClass}__groupname`}>{groupname}</td>
      </tr>
    );
  }
}

export default HostUsersListRow;
