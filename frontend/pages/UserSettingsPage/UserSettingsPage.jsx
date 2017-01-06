import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { goBack } from 'react-router-redux';
import moment from 'moment';

import Avatar from 'components/Avatar';
import Button from 'components/buttons/Button';
import ChangePasswordForm from 'components/forms/ChangePasswordForm';
import Icon from 'components/icons/Icon';
import { logoutUser, updateUser } from 'redux/nodes/auth/actions';
import Modal from 'components/modals/Modal';
import { renderFlash } from 'redux/nodes/notifications/actions';
import userInterface from 'interfaces/user';
import UserSettingsForm from 'components/forms/UserSettingsForm';

const baseClass = 'user-settings';

class UserSettingsPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func.isRequired,
    errors: PropTypes.shape({
      username: PropTypes.string,
      base: PropTypes.string,
    }),
    user: userInterface,
  };

  constructor (props) {
    super(props);

    this.state = { showModal: false };
  }

  onCancel = (evt) => {
    evt.preventDefault();

    const { dispatch } = this.props;

    dispatch(goBack());

    return false;
  }

  onLogout = (evt) => {
    evt.preventDefault();

    const { dispatch } = this.props;

    dispatch(logoutUser());

    return false;
  }

  onShowModal = (evt) => {
    evt.preventDefault();

    this.setState({ showModal: true });

    return false;
  }

  onToggleModal = (evt) => {
    evt.preventDefault();

    const { showModal } = this.state;

    this.setState({ showModal: !showModal });

    return false;
  }

  handleSubmit = (formData) => {
    const { dispatch, user } = this.props;

    return dispatch(updateUser(user, formData))
      .then(() => {
        return dispatch(renderFlash('success', 'Account updated!'));
      })
      .catch(() => false);
  }

  handleSubmitPasswordForm = (formData) => {
    console.log('Change Password Form submitted', formData);

    this.setState({ showModal: false });

    return false;
  }

  renderModal = () => {
    const { showModal } = this.state;
    const { handleSubmitPasswordForm, onToggleModal } = this;

    if (!showModal) {
      return false;
    }

    return (
      <Modal
        title="Change Password"
        onExit={onToggleModal}
      >
        <ChangePasswordForm
          handleSubmit={handleSubmitPasswordForm}
          onCancel={onToggleModal}
        />
      </Modal>
    );
  }

  render () {
    const { handleSubmit, onCancel, onLogout, onShowModal, renderModal } = this;
    const { errors, user } = this.props;

    if (!user) {
      return false;
    }

    const { updated_at: updatedAt } = user;
    const lastUpdatedAt = moment(updatedAt).fromNow();

    return (
      <div className={baseClass}>
        <div className={`${baseClass}__manage body-wrap`}>
          <h1>Manage User Settings</h1>
          <UserSettingsForm
            formData={user}
            handleSubmit={handleSubmit}
            onCancel={onCancel}
            serverErrors={errors}
          />
        </div>
        <div className={`${baseClass}__additional body-wrap`}>
          <h1>Additional Info</h1>

          <div className={`${baseClass}__change-avatar`}>
            <Avatar user={user} className={`${baseClass}__avatar`} />
            <a href="http://en.gravatar.com/emails/">Change Photo at Gravatar</a>
          </div>

          <div className={`${baseClass}__more-info-detail`}>
            <Icon name="username" />
            <strong>Role</strong> - USER
          </div>
          <div className={`${baseClass}__more-info-detail`}>
            <Icon name="lock-big" />
            <strong>Password</strong>
          </div>
          <Button onClick={onShowModal} variant="brand" className={`${baseClass}__button`}>
            CHANGE PASSWORD
          </Button>
          <p className={`${baseClass}__last-updated`}>Last changed: {lastUpdatedAt}</p>
          <Button onClick={onLogout} variant="alert" className={`${baseClass}__button`}>
            LOGOUT
          </Button>
        </div>
        {renderModal()}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { errors, user } = state.auth;

  return { errors, user };
};

export default connect(mapStateToProps)(UserSettingsPage);
