import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { goBack } from 'react-router-redux';
import moment from 'moment';

import Avatar from 'components/Avatar';
import Button from 'components/buttons/Button';
import ChangePasswordForm from 'components/forms/ChangePasswordForm';
import Icon from 'components/Icon';
import { logoutUser } from 'redux/nodes/auth/actions';
import Modal from 'components/modals/Modal';
import { renderFlash } from 'redux/nodes/notifications/actions';
import userActions from 'redux/nodes/entities/users/actions';
import userInterface from 'interfaces/user';
import UserSettingsForm from 'components/forms/UserSettingsForm';

const baseClass = 'user-settings-page';

class UserSettingsPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func.isRequired,
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
    const { update } = userActions;

    return dispatch(update(user, formData))
      .then(() => {
        return dispatch(renderFlash('success', 'Account updated!'));
      });
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
    const { user } = this.props;

    if (!user) {
      return false;
    }

    const { updated_at: updatedAt } = user;
    const lastUpdatedAt = moment(updatedAt).fromNow();

    return (
      <div>
        <div className="body-wrap">
          <h1>Manage User Settings</h1>
          <UserSettingsForm formData={user} handleSubmit={handleSubmit} onCancel={onCancel} />
        </div>
        <div className="body-wrap">
          <h1>Additional Info</h1>
          <Avatar user={user} />
          <div className={`${baseClass}__change-avatar-text`}>
            Change Photo at Gravatar
          </div>
          <div className={`${baseClass}__more-info-detail`}>
            <Icon name="username" />
            <b>Role</b> - USER
          </div>
          <div className={`${baseClass}__more-info-detail`}>
            <Icon name="lock-big" />
            <b>Password</b>
          </div>
          <Button onClick={onShowModal} text="CHANGE PASSWORD" variant="brand" />
          <small>Last changed: {lastUpdatedAt}</small>
          <Button onClick={onLogout} text="LOGOUT" variant="alert" />
        </div>
        {renderModal()}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { user } = state.auth;

  return { user };
};

export default connect(mapStateToProps)(UserSettingsPage);
