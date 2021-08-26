import React, { Component } from "react";
import hostUserInterface from "interfaces/host_users";

const baseClass = "hosts-user-list-row";

class HostUsersListRow extends Component {
  static propTypes = {
    hostUser: hostUserInterface.isRequired,
  };

  render() {
    const { hostUser } = this.props;
    const { username } = hostUser;

    return (
      <tr>
        <td className={`${baseClass}__username`}>{username}</td>
      </tr>
    );
  }
}

export default HostUsersListRow;
