import React, { useState, useContext, useEffect } from "react";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { IUser } from "interfaces/user";
import usersAPI from "services/entities/users";
import { authToken } from "utilities/local";
import deepDifference from "utilities/deep_difference";
import formatErrorResponse from "utilities/format_error_response";

import Button from "components/buttons/Button";
// @ts-ignore
import ChangeEmailForm from "components/forms/ChangeEmailForm";
// @ts-ignore
import ChangePasswordForm from "components/forms/ChangePasswordForm";
// @ts-ignore
import Modal from "components/Modal";

// @ts-ignore
import UserSettingsForm from "components/forms/UserSettingsForm";
import InfoBanner from "components/InfoBanner";
import SecretField from "components/SecretField";
import SandboxGate from "components/Sandbox/SandboxGate";
import SandboxDemoMessage from "components/Sandbox/SandboxDemoMessage";
import MainContent from "components/MainContent";
import SidePanelContent from "components/SidePanelContent";

import ExternalURLIcon from "../../../assets/images/icon-external-url-12x12@2x.png";
import UserSidePanel from "./UserSidePanel";

const baseClass = "user-settings";

interface IUserSettingsPageProps {
  router: InjectedRouter;
}

const UserSettingsPage = ({
  router,
}: IUserSettingsPageProps): JSX.Element | null => {
  const { config, currentUser, isSandboxMode } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const [pendingEmail, setPendingEmail] = useState<string>("");
  const [showEmailModal, setShowEmailModal] = useState<boolean>(false);
  const [showPasswordModal, setShowPasswordModal] = useState<boolean>(false);
  const [updatedUser, setUpdatedUser] = useState<Partial<IUser>>({});
  const [showApiTokenModal, setShowApiTokenModal] = useState<boolean>(false);
  const [errors, setErrors] = useState<{ [key: string]: string }>({});
  const [userErrors, setUserErrors] = useState<{ [key: string]: string }>({});

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

  const renderApiTokenModal = () => {
    if (!showApiTokenModal) {
      return false;
    }

    return (
      <Modal
        title="Get API token"
        onExit={onToggleApiTokenModal}
        onEnter={onToggleApiTokenModal}
      >
        <>
          <InfoBanner>
            <p>
              <strong>This token expires.</strong> If you want an API key for a
              permanent integration, create an&nbsp;
              <a
                href="https://fleetdm.com/docs/using-fleet/fleetctl-cli#using-fleetctl-with-an-api-only-user?utm_medium=fleetui&utm_campaign=get-api-token"
                target="_blank"
                rel="noopener noreferrer"
              >
                API-only user
              </a>
              <img
                alt="Open external link"
                className="icon-external"
                src={ExternalURLIcon}
              />
              &nbsp;instead.
            </p>
          </InfoBanner>
          <div className={`${baseClass}__secret-wrapper`}>
            <SecretField secret={authToken()} />
          </div>
          <p className="token-message">
            This token is intended for SSO users to authenticate in the fleetctl
            CLI. It expires based on the{" "}
            <a
              href="https://fleetdm.com/docs/deploying/configuration#session-duration?utm_medium=fleetui&utm_campaign=get-api-token"
              target="_blank"
              rel="noopener noreferrer"
            >
              session duration configuration
            </a>
            <img
              alt="Open external link"
              className="icon-external"
              src={ExternalURLIcon}
            />
          </p>
          <div className="modal-cta-wrap">
            <Button onClick={onToggleApiTokenModal} type="button">
              Done
            </Button>
          </div>
        </>
      </Modal>
    );
  };

  if (!currentUser) {
    return null;
  }

  return (
    <>
      <MainContent className={baseClass}>
        <SandboxGate
          fallbackComponent={() => (
            <SandboxDemoMessage
              className={`${baseClass}__sandboxMode`}
              message="Account management is only available in self-managed Fleet"
              utmSource="fleet-ui-my-account-page"
            />
          )}
        >
          <div className={`${baseClass}__manage`}>
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
          {renderEmailModal()}
          {renderPasswordModal()}
          {renderApiTokenModal()}
        </SandboxGate>
      </MainContent>
      <SandboxGate>
        <SidePanelContent>
          <UserSidePanel
            currentUser={currentUser}
            onChangePassword={onShowPasswordModal}
            onGetApiToken={onShowApiTokenModal}
          />
        </SidePanelContent>
      </SandboxGate>
    </>
  );
};

export default UserSettingsPage;
