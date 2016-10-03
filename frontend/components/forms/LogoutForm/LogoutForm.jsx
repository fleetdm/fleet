import React, { Component, PropTypes } from 'react';
import componentStyles from './styles';
import Button from '../../buttons/Button';

class LogoutForm extends Component {
  static propTypes = {
    onSubmit: PropTypes.func,
    user: PropTypes.object,
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
      submitButtonStyles,
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
          onClick={onFormSubmit}
          style={submitButtonStyles}
          text="Logout"
          type="submit"
          variant="gradient"
        />
      </form>
    );
  }
}

export default LogoutForm;
