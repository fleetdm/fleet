import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { goBack } from "react-router-redux";
import moment from "moment";
import { authToken } from "utilities/local";
import { stringToClipboard } from "utilities/copy_text";

import { noop } from "lodash";

import Avatar from "components/Avatar";
import Button from "components/buttons/Button";
import ChangeEmailForm from "components/forms/ChangeEmailForm";
import ChangePasswordForm from "components/forms/ChangePasswordForm";
import deepDifference from "utilities/deep_difference";
import permissionUtils from "utilities/permissions";
import FleetIcon from "components/icons/FleetIcon";
import InputField from "components/forms/fields/InputField";
import { logoutUser, updateUser } from "redux/nodes/auth/actions";
import Modal from "components/Modal";
import configInterface from "interfaces/config";
import versionInterface from "interfaces/version";
import { renderFlash } from "redux/nodes/notifications/actions";
import userActions from "redux/nodes/entities/users/actions";
import versionActions from "redux/nodes/version/actions";
import userInterface from "interfaces/user";
import UserSettingsForm from "components/forms/UserSettingsForm";
import { generateRole, generateTeam, greyCell } from "fleet/helpers";

const baseClass = "user-settings";

export class UserSettingsPage extends Component {
  static propTypes = {
    config: configInterface,
    dispatch: PropTypes.func.isRequired,
    version: versionInterface,
    errors: PropTypes.shape({
      email: PropTypes.string,
      base: PropTypes.string,
    }),
    user: userInterface,
    userErrors: PropTypes.shape({
      base: PropTypes.string,
      new_password: PropTypes.string,
      old_password: PropTypes.string,
    }),
    isPremiumTier: PropTypes.bool,
  };

  static defaultProps = {
    version: {},
    dispatch: noop,
  };

  constructor(props) {
    super(props);

    this.state = {
      pendingEmail: undefined,
      showEmailModal: false,
      showPasswordModal: false,
      updatedUser: {},
      copyMessage: "",
    };
  }

  componentDidMount() {
    const { dispatch } = this.props;

    dispatch(versionActions.getVersion());

    return false;
  }

  onCancel = (evt) => {
    evt.preventDefault();

    const { dispatch } = this.props;

    dispatch(goBack());

    return false;
  };

  onLogout = (evt) => {
    evt.preventDefault();

    const { dispatch } = this.props;

    dispatch(logoutUser());

    return false;
  };

  onShowModal = (evt) => {
    evt.preventDefault();

    this.setState({ showPasswordModal: true });

    return false;
  };

  onShowApiTokenModal = (evt) => {
    evt.preventDefault();

    this.setState({ showApiTokenModal: true });

    return false;
  };

  onToggleEmailModal = (updatedUser = {}) => {
    const { showEmailModal } = this.state;

    this.setState({
      showEmailModal: !showEmailModal,
      updatedUser,
    });

    return false;
  };

  onTogglePasswordModal = (evt) => {
    evt.preventDefault();

    const { showPasswordModal } = this.state;

    this.setState({ showPasswordModal: !showPasswordModal });

    return false;
  };

  onToggleApiTokenModal = (evt) => {
    evt.preventDefault();

    const { showApiTokenModal } = this.state;

    this.setState({ showApiTokenModal: !showApiTokenModal });

    return false;
  };

  onToggleSecret = (evt) => {
    evt.preventDefault();

    const { revealSecret } = this.state;

    this.setState({ revealSecret: !revealSecret });
    return false;
  };

  onCopySecret = () => {
    return (evt) => {
      evt.preventDefault();

      stringToClipboard(authToken())
        .then(() => this.setState({ copyMessage: "Copied!" }))
        .catch(() => this.setState({ copyMessage: "Copy failed" }));

      // Clear message after 1 second
      setTimeout(() => this.setState({ copyMessage: "" }), 1000);

      return false;
    };
  };

  handleSubmit = (formData) => {
    const { dispatch, user, config } = this.props;
    const updatedUser = deepDifference(formData, user);

    if (updatedUser.email && !updatedUser.password) {
      return this.onToggleEmailModal(updatedUser);
    }

    return dispatch(updateUser(user, updatedUser))
      .then(() => {
        let accountUpdatedFlashMessage = "Account updated";
        if (updatedUser.email) {
          accountUpdatedFlashMessage += `: A confirmation email was sent from ${config.sender_address} to ${updatedUser.email}`;
          this.setState({ pendingEmail: updatedUser.email });
        }

        dispatch(renderFlash("success", accountUpdatedFlashMessage));

        return true;
      })
      .catch(() => false);
  };

  handleSubmitPasswordForm = (formData) => {
    const { dispatch, user } = this.props;

    return dispatch(userActions.changePassword(user, formData)).then(() => {
      dispatch(renderFlash("success", "Password changed successfully"));
      this.setState({ showPasswordModal: false });

      return false;
    });
  };

  renderEmailModal = () => {
    const { errors } = this.props;
    const { updatedUser, showEmailModal } = this.state;
    const { handleSubmit, onToggleEmailModal } = this;

    const emailSubmit = (formData) => {
      handleSubmit(formData).then((r) => {
        return r ? onToggleEmailModal() : false;
      });
    };

    if (!showEmailModal) {
      return false;
    }

    return (
      <Modal title="Confirm email update" onExit={onToggleEmailModal}>
        <div className={`${baseClass}__confirm-update`}>
          To update your email you must confirm your password.
        </div>
        <ChangeEmailForm
          formData={updatedUser}
          handleSubmit={emailSubmit}
          onCancel={onToggleEmailModal}
          serverErrors={errors}
        />
      </Modal>
    );
  };

  renderPasswordModal = () => {
    const { userErrors } = this.props;
    const { showPasswordModal } = this.state;
    const { handleSubmitPasswordForm, onTogglePasswordModal } = this;

    if (!showPasswordModal) {
      return false;
    }

    return (
      <Modal title="Change password" onExit={onTogglePasswordModal}>
        <ChangePasswordForm
          handleSubmit={handleSubmitPasswordForm}
          onCancel={onTogglePasswordModal}
          serverErrors={userErrors}
        />
      </Modal>
    );
  };

  renderLabel = () => {
    const { copyMessage } = this.state;
    const { onCopySecret } = this;

    return (
      <span className={`${baseClass}__name`}>
        <span className="buttons">
          {copyMessage && <span>{`${copyMessage} `}</span>}
          <Button
            variant="unstyled"
            className={`${baseClass}__secret-copy-icon`}
            onClick={onCopySecret(`.${baseClass}__secret-input`)}
          >
            <FleetIcon name="clipboard" />
          </Button>
        </span>
      </span>
    );
  };

  renderApiTokenModal = () => {
    const { showApiTokenModal, revealSecret } = this.state;
    const { onToggleApiTokenModal, onToggleSecret, renderLabel } = this;

    if (!showApiTokenModal) {
      return false;
    }

    return (
      <Modal title="Get API token" onExit={onToggleApiTokenModal}>
        <p className={`${baseClass}__secret-label`}>
          Your API Token:
          <a
            href="#revealSecret"
            onClick={onToggleSecret}
            className={`${baseClass}__reveal-secret`}
          >
            {revealSecret ? "Hide" : "Reveal"} Token
          </a>
        </p>
        <div className={`${baseClass}__secret-wrapper`}>
          <InputField
            disabled
            inputWrapperClass={`${baseClass}__secret-input`}
            name="osqueryd-secret"
            type={revealSecret ? "text" : "password"}
            value={authToken()}
            label={renderLabel()}
          />
        </div>
        <div className={`${baseClass}__button-wrap`}>
          <Button
            onClick={onToggleApiTokenModal}
            className="button button--brand"
          >
            Done
          </Button>
        </div>
      </Modal>
    );
  };

  render() {
    const {
      handleSubmit,
      onCancel,
      onShowModal,
      onShowApiTokenModal,
      renderEmailModal,
      renderPasswordModal,
      renderApiTokenModal,
    } = this;
    const { version, errors, user, config, isPremiumTier } = this.props;
    const { pendingEmail } = this.state;

    if (!user) {
      return false;
    }

    const {
      global_role: globalRole,
      updated_at: updatedAt,
      sso_enabled: ssoEnabled,
      teams,
    } = user;

    const roleText = generateRole(teams, globalRole);
    const teamsText = generateTeam(teams, globalRole);

    const lastUpdatedAt = moment(updatedAt).fromNow();

    return (
      <div className={baseClass}>
        <div className={`${baseClass}__manage body-wrap`}>
          <h1>My account</h1>
          <UserSettingsForm
            formData={user}
            handleSubmit={handleSubmit}
            onCancel={onCancel}
            pendingEmail={pendingEmail}
            serverErrors={errors}
            smtpConfigured={config.configured}
          />
        </div>
        <div className={`${baseClass}__additional body-wrap`}>
          <div className={`${baseClass}__change-avatar`}>
            <Avatar user={user} className={`${baseClass}__avatar`} />
            <a href="http://en.gravatar.com/emails/">
              Change photo at Gravatar
            </a>
          </div>
          {isPremiumTier && (
            <div className={`${baseClass}__more-info-detail`}>
              <p className={`${baseClass}__header`}>Teams</p>
              <p
                className={`${baseClass}__description ${baseClass}__teams ${greyCell(
                  teamsText
                )}`}
              >
                {teamsText}
              </p>
            </div>
          )}
          <div className={`${baseClass}__more-info-detail`}>
            <p className={`${baseClass}__header`}>Role</p>
            <p
              className={`${baseClass}__description ${baseClass}__role ${greyCell(
                roleText
              )}`}
            >
              {roleText}
            </p>
          </div>
          <div className={`${baseClass}__more-info-detail`}>
            <p className={`${baseClass}__header`}>Password</p>
          </div>
          <Button
            onClick={onShowModal}
            disabled={ssoEnabled}
            className={`${baseClass}__button`}
          >
            Change password
          </Button>
          <p className={`${baseClass}__last-updated`}>
            Last changed: {lastUpdatedAt}
          </p>
          <Button
            onClick={onShowApiTokenModal}
            className={`${baseClass}__button`}
          >
            Get API token
          </Button>
          <span
            className={`${baseClass}__version`}
          >{`Fleet ${version.version} • Go ${version.go_version}`}</span>
        </div>
        {renderEmailModal()}
        {renderPasswordModal()}
        {renderApiTokenModal()}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { data: version } = state.version;
  const { errors, user } = state.auth;
  const { config } = state.app;
  const { errors: userErrors } = state.entities.users;
  const isPremiumTier = permissionUtils.isPremiumTier(config);

  return { version, errors, user, userErrors, config, isPremiumTier };
};

export default connect(mapStateToProps)(UserSettingsPage);
