import React, { Component, PropTypes } from 'react';

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
    const { user } = this.props;
    const { gravatarURL } = user;
    const { onFormSubmit } = this;

    return (
      <form onSubmit={onFormSubmit} className={baseClass}>
        <div className={`${baseClass}__container`}>
          <img alt="Avatar" src={gravatarURL} className={`${baseClass}__avatar`} />
          <p className={`${baseClass}__username`}>{user.username}</p>
          <p className={`${baseClass}__subtext`}>Are you sure you want to log out?</p>
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
