import React, { Component } from "react";
import PropTypes from "prop-types";

import PATHS from "router/paths";

import DropdownButton from "components/buttons/DropdownButton";
import Avatar from "../../Avatar";

class UserMenu extends Component {
  static propTypes = {
    onLogout: PropTypes.func,
    onNavItemClick: PropTypes.func,
    user: PropTypes.shape({
      gravatarURL: PropTypes.string,
      name: PropTypes.string,
      email: PropTypes.string.isRequired,
      position: PropTypes.string,
    }).isRequired,
  };

  static defaultProps = {
    isOpened: false,
  };

  constructor(props) {
    super(props);

    const accountNavigate = props.onNavItemClick(PATHS.USER_SETTINGS);
    this.dropdownItems = [
      {
        label: "My account",
        onClick: accountNavigate,
      },
      {
        label: "Documentation",
        onClick: () =>
          window.open(
            "https://github.com/fleetdm/fleet/blob/main/docs/README.md",
            "_blank"
          ),
      },
      {
        label: "Sign out",
        onClick: props.onLogout,
      },
    ];
  }

  render() {
    const { user } = this.props;
    const baseClass = "user-menu";

    return (
      <div className={baseClass}>
        <DropdownButton options={this.dropdownItems}>
          <Avatar
            className={`${baseClass}__avatar-image`}
            user={user}
            size="small"
          />
        </DropdownButton>
      </div>
    );
  }
}

export default UserMenu;
