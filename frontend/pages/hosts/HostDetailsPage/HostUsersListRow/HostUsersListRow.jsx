import React, { Component } from "react";

const baseClass = "hosts-user-list-row";

class HostUsersListRow extends Component {
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
