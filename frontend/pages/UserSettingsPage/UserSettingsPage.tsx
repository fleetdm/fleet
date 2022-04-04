import React, { useState, useContext, useEffect } from "react";
import { InjectedRouter } from "react-router";
import { formatDistanceToNow } from "date-fns";
import { authToken } from "utilities/local"; // @ts-ignore
import { stringToClipboard } from "utilities/copy_text";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification"; // @ts-ignore
import { IVersionData } from "interfaces/version";
import { IUser } from "interfaces/user"; // @ts-ignore
import deepDifference from "utilities/deep_difference";
import usersAPI from "services/entities/users";
import versionAPI from "services/entities/version"; // @ts-ignore
import { formatErrorResponse } from "redux/nodes/entities/base/helpers";
import { generateRole, generateTeam, greyCell } from "fleet/helpers";

import Avatar from "components/Avatar";
import Button from "components/buttons/Button"; // @ts-ignore
import ChangeEmailForm from "components/forms/ChangeEmailForm"; // @ts-ignore
import ChangePasswordForm from "components/forms/ChangePasswordForm"; // @ts-ignore
import FleetIcon from "components/icons/FleetIcon"; // @ts-ignore
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal"; // @ts-ignore
import UserSettingsForm from "components/forms/UserSettingsForm";

const baseClass = "user-settings";

interface IUserSettingsPageProps {
  router: InjectedRouter;
}

const UserSettingsPage = ({ router }: IUserSettingsPageProps) => {
  const { config, currentUser, isPremiumTier } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);
  const [pendingEmail, setPendingEmail] = useState<string>("");
  const [showEmailModal, setShowEmailModal] = useState<boolean>(false);
  const [showPasswordModal, setShowPasswordModal] = useState<boolean>(false);
  const [updatedUser, setUpdatedUser] = useState<Partial<IUser>>({});
  const [copyMessage, setCopyMessage] = useState<string>("");
  const [showApiTokenModal, setShowApiTokenModal] = useState<boolean>(false);
  const [revealSecret, setRevealSecret] = useState<boolean>(false);
  const [versionData, setVersionData] = useState<IVersionData>();
  const [errors, setErrors] = useState<{ [key: string]: string }>({});
  const [userErrors, setUserErrors] = useState<{ [key: string]: string }>({});

  useEffect(() => {
    const getVersionData = async () => {
      try {
        const data = await versionAPI.load();
        setVersionData(data);
      } catch (response) {
        console.error(response);
        return false;
      }
    };

    getVersionData();
  }, []);

  const onCancel = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    return router.goBack();
  };

  const onShowPasswordModal = () => {
    setShowPasswordModal(true);
    return false;
  };

  const onShowApiTokenModal = () => {
    setShowApiTokenModal(true);
    return false;
  };

  const onToggleEmailModal = (updated = {}) => {
    setShowEmailModal(!showEmailModal);
    setUpdatedUser(updated);
    return false;
  };

  const onTogglePasswordModal = () => {
    setShowPasswordModal(!showPasswordModal);
    return false;
  };

  const onToggleApiTokenModal = () => {
    setShowApiTokenModal(!showApiTokenModal);
    return false;
  };

  const onToggleSecret = () => {
    setRevealSecret(!revealSecret);
    return false;
  };

  // placeholder is needed even though it's not used
  const onCopySecret = (placeholder: string) => {
    return (evt: ClipboardEvent) => {
      evt.preventDefault();

      stringToClipboard(authToken())
        .then(() => setCopyMessage("Copied!"))
        .catch(() => setCopyMessage("Copy failed"));

      // Clear message after 1 second
      setTimeout(() => setCopyMessage(""), 1000);
      return false;
    };
  };

  const handleSubmit = async (formData: any) => {
    if (!currentUser) {
      return false;
    }

    const updated = deepDifference(formData, currentUser);

    if (updated.email && !updated.password) {
      return onToggleEmailModal(updated);
    }

    try {
      await usersAPI.update(currentUser.id, updated);
      let accountUpdatedFlashMessage = "Account updated";
      if (updated.email) {
        accountUpdatedFlashMessage += `: A confirmation email was sent from ${config?.smtp_settings.sender_address} to ${updated.email}`;
        setPendingEmail(updated.email);
      }

      renderFlash("success", accountUpdatedFlashMessage);
      return true;
    } catch (response) {
      const errorObject = formatErrorResponse(response);
      setErrors(errorObject);

      if (errorObject.base.includes("already exists")) {
        renderFlash("error", "A user with this email address already exists.");
      } else {
        renderFlash("error", "Could not edit user. Please try again.");
      }

      setShowEmailModal(false);
      return false;
    }
  };

  const handleSubmitPasswordForm = async (formData: any) => {
    try {
      await usersAPI.changePassword(formData);
      renderFlash("success", "Password changed successfully");
      setShowPasswordModal(false);
    } catch (response) {
      const errorObject = formatErrorResponse(response);
      setUserErrors(errorObject);
      return false;
    }
  };

  const renderEmailModal = () => {
    const emailSubmit = (formData: any) => {
      handleSubmit(formData).then((r?: boolean) => {
        return r ? onToggleEmailModal() : false;
      });
    };

    if (!showEmailModal) {
      return false;
    }

    return (
      <Modal title="Confirm email update" onExit={onToggleEmailModal}>
        <>
          <div className={`${baseClass}__confirm-update`}>
            To update your email you must confirm your password.
          </div>
          <ChangeEmailForm
            formData={updatedUser}
            handleSubmit={emailSubmit}
            onCancel={onToggleEmailModal}
            serverErrors={errors}
          />
        </>
      </Modal>
    );
  };

  const renderPasswordModal = () => {
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

  const renderLabel = () => {
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

  const renderApiTokenModal = () => {
    if (!showApiTokenModal) {
      return false;
    }

    return (
      <Modal title="Get API token" onExit={onToggleApiTokenModal}>
        <>
          <p className={`${baseClass}__secret-label`}>
            Your API token:
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
        </>
      </Modal>
    );
  };

  if (!currentUser) {
    return false;
  }

  const {
    global_role: globalRole,
    updated_at: updatedAt,
    sso_enabled: ssoEnabled,
    teams,
  } = currentUser;

  const roleText = generateRole(teams, globalRole);
  const teamsText = generateTeam(teams, globalRole);

  const lastUpdatedAt =
    updatedAt &&
    formatDistanceToNow(new Date(updatedAt), {
      addSuffix: true,
    });

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__manage body-wrap`}>
        <h1>My account</h1>
        <UserSettingsForm
          formData={currentUser}
          handleSubmit={handleSubmit}
          onCancel={onCancel}
          pendingEmail={pendingEmail}
          serverErrors={errors}
          smtpConfigured={config?.smtp_settings.configured}
        />
      </div>
      <div className={`${baseClass}__additional body-wrap`}>
        <div className={`${baseClass}__change-avatar`}>
          <Avatar user={currentUser} className={`${baseClass}__avatar`} />
          <a href="http://en.gravatar.com/emails/">Change photo at Gravatar</a>
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
          onClick={onShowPasswordModal}
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
        >{`Fleet ${versionData?.version} • Go ${versionData?.go_version}`}</span>
        <span className={`${baseClass}__privacy-policy`}>
          <a
            href="https://fleetdm.com/legal/privacy"
            target="_blank"
            rel="noopener noreferrer"
          >
            Privacy policy
          </a>
        </span>
      </div>
      {renderEmailModal()}
      {renderPasswordModal()}
      {renderApiTokenModal()}
    </div>
  );
};

export default UserSettingsPage;
