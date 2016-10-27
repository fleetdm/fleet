import React, { Component, PropTypes } from 'react';

import componentStyles from './styles';
import Button from '../../buttons/Button';
import userInterface from '../../../interfaces/user';

const baseClass = 'logout-form';

class LogoutForm extends Component {
  static propTypes = {
    onSubmit: PropTypes.func,
    user: userInterface,
  };

  onFormSubmit = (evt) => {
    evt.preventDefault();

    const { onSubmit } = this.props;

    return onSubmit();
  }

  render () {
    const {
      avatarStyles,
      containerStyles,
      formStyles,
      subtextStyles,
      usernameStyles,
    } = componentStyles;
    const { user } = this.props;
    const { gravatarURL } = user;
    const { onFormSubmit } = this;

    return (
      <form onSubmit={onFormSubmit} style={formStyles}>
        <div style={containerStyles}>
          <img alt="Avatar" src={gravatarURL} style={avatarStyles} />
          <p style={usernameStyles}>{user.username}</p>
          <p style={subtextStyles}>Are you sure you want to log out?</p>
        </div>
        <Button
          className={`${baseClass}__submit-btn`}
          onClick={onFormSubmit}
          text="Logout"
          type="submit"
          variant="gradient"
        />
      </form>
    );
  }
}

export default LogoutForm;
