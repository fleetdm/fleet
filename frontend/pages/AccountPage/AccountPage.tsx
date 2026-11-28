import { upperFirst } from "lodash";
import React, { useState, useContext } from "react";
import { InjectedRouter } from "react-router";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import ChangeEmailForm from "components/forms/ChangeEmailForm";
import ChangePasswordForm from "components/forms/ChangePasswordForm";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import UserSettingsForm from "components/forms/UserSettingsForm";
import InfoBanner from "components/InfoBanner";
import MainContent from "components/MainContent";
// @ts-ignore
import Modal from "components/Modal";
import SidePanelContent from "components/SidePanelContent";
import SidePanelPage from "components/SidePanelPage";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import { splitServerErrors } from "interfaces/errors";
import { IUser } from "interfaces/user";
import usersAPI from "services/entities/users";
import authToken from "utilities/auth_token";
import deepDifference from "utilities/deep_difference";
import formatErrorResponse from "utilities/format_error_response";

import AccountSidePanel from "./AccountSidePanel";
import { getErrorMessage, INCORRECT_PASSWORD_ERROR } from "./helpers";

const baseClass = "account-page";

interface IAccountPageProps {
  router: InjectedRouter;
}

const AccountPage = ({ router }: IAccountPageProps): JSX.Element | null => {
  const { config, currentUser } = useContext(AppContext);

  const [pendingEmail, setPendingEmail] = useState("");
  const [showEmailModal, setShowEmailModal] = useState(false);
  const [showPasswordModal, setShowPasswordModal] = useState(false);
  const [updatedUser, setUpdatedUser] = useState<Partial<IUser>>({});
  const [showApiTokenModal, setShowApiTokenModal] = useState(false);
  const [errors, setErrors] = useState<{ [key: string]: string }>({});
  // Kept apart from `errors` (which the main form also reads) so a wrong
  // password isn't toasted a second time by the account form behind the modal.
  const [emailModalErrors, setEmailModalErrors] = useState<{
    [key: string]: string;
  }>({});
  const [passwordErrors, setPasswordErrors] = useState<{
    [key: string]: string;
  }>({});

  const onCancel = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    return router.goBack();
  };

  const onShowPasswordModal = () => {
    setShowPasswordModal(true);
    // Drop stale server errors so a reopen doesn't replay a previous attempt's
    // toast through the form hook's serverErrors effect.
    setPasswordErrors({});
    return false;
  };

  const onShowApiTokenModal = () => {
    setShowApiTokenModal(true);
    return false;
  };

  const onToggleEmailModal = (updated = {}) => {
    setShowEmailModal(!showEmailModal);
    setUpdatedUser(updated);
    // Drop stale server errors so a reopen doesn't replay a previous attempt's
    // toast through the form hook's serverErrors effect.
    setErrors({});
    setEmailModalErrors({});
    return false;
  };

  const onTogglePasswordModal = () => {
    setShowPasswordModal(!showPasswordModal);
    // Drop stale server errors so a reopen doesn't replay a previous attempt's
    // toast through the form hook's serverErrors effect.
    setPasswordErrors({});
    return false;
  };

  const onToggleApiTokenModal = () => {
    setShowApiTokenModal(!showApiTokenModal);
    return false;
  };

  const handleSubmit = async (formData: Partial<IUser>) => {
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
        // always present, this for typing
        const senderAddressMessage = config?.smtp_settings?.sender_address
          ? ` from ${config?.smtp_settings?.sender_address}`
          : "";
        accountUpdatedFlashMessage += `: A confirmation email was sent${senderAddressMessage} to ${updated.email}`;
        setPendingEmail(updated.email);
      }

      notify.success(accountUpdatedFlashMessage);
      return true;
    } catch (response) {
      const errorObject = formatErrorResponse(response);
      // formatErrorResponse always includes `base` and `http_status`; the forms
      // only want inline, field-keyed messages, so strip the globals before
      // passing them through (the toast below already covers the global error).
      const {
        base: _base,
        http_status: httpStatus,
        ...fieldErrors
      } = errorObject;

      // A wrong password in the Confirm email update modal comes back as a bare
      // 403 with no `errors` array, so there is no `base` or field message to
      // read. Show it on the password field (the form hook also toasts it) and
      // keep the modal open so the user can retry.
      if (
        updated.email &&
        updated.password &&
        httpStatus === 403 &&
        !errorObject.base
      ) {
        setEmailModalErrors({ password: INCORRECT_PASSWORD_ERROR });
        return false;
      }

      setErrors(fieldErrors);
      notify.error(
        errorObject.base?.includes("already exists")
          ? "A user with this email address already exists."
          : "Could not edit user. Please try again.",
        { response }
      );

      setShowEmailModal(false);
      return false;
    }
  };

  const handleSubmitPasswordForm = async (formData: {
    old_password: string;
    new_password: string;
  }) => {
    try {
      await usersAPI.changePassword(formData);
      notify.success("Password changed successfully");
      setShowPasswordModal(false);
      setPasswordErrors({});
    } catch (e) {
      // A field-specific error (e.g. a wrong current password) renders inline
      // on its field; the form hook also toasts it. Anything else is toasted
      // here.
      const { fieldErrors } = splitServerErrors(e, {
        old_password: "old_password",
      });
      if (fieldErrors.old_password) {
        // The server's reason is lowercase API output ("old password does not
        // match"); capitalize it to match the rest of the UI copy.
        setPasswordErrors({
          old_password: upperFirst(fieldErrors.old_password),
        });
      } else {
        notify.error(getErrorMessage(e), { response: e });
      }
    }
  };

  const renderEmailModal = () => {
    // Return the promise so useFormValidation can track in-flight state and
    // disable the password field + Submit button until the request resolves.
    const emailSubmit = (formData: Partial<IUser>) =>
      handleSubmit(formData).then((r?: boolean) => {
        if (r) {
          onToggleEmailModal();
        }
      });

    if (!showEmailModal) {
      return false;
    }

    return (
      <Modal title="Confirm email update" onExit={onToggleEmailModal}>
        <ChangeEmailForm
          defaultData={updatedUser}
          handleSubmit={emailSubmit}
          onCancel={onToggleEmailModal}
          serverErrors={emailModalErrors}
        />
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
          serverErrors={passwordErrors}
        />
      </Modal>
    );
  };

  const renderApiTokenModal = () => {
    if (!showApiTokenModal) {
      return false;
    }

    // TODO - move to its own component
    return (
      <Modal
        title="Get API token"
        onExit={onToggleApiTokenModal}
        onEnter={onToggleApiTokenModal}
      >
        <InfoBanner>
          <p>
            <strong>This token expires.</strong> If you want an API key for a
            permanent integration, create an&nbsp;
            <CustomLink
              url="https://fleetdm.com/docs/using-fleet/fleetctl-cli?utm_medium=fleetui&utm_campaign=get-api-token#using-fleetctl-with-an-api-only-user"
              text="API-only user"
              newTab
              variant="banner-link"
            />
            &nbsp;instead.
          </p>
        </InfoBanner>
        <InputFieldHiddenContent
          value={authToken.get() || ""}
          helpText={
            <>
              This token is intended for SSO users to authenticate in the
              fleetctl CLI. It expires based on the{" "}
              <CustomLink
                url="https://fleetdm.com/docs/deploying/configuration?utm_medium=fleetui&utm_campaign=get-api-token#session-duration"
                text="session duration configuration"
                newTab
              />
            </>
          }
        />
        <div className="modal-cta-wrap">
          <Button onClick={onToggleApiTokenModal} type="button">
            Close
          </Button>
        </div>
      </Modal>
    );
  };

  if (!currentUser) {
    return null;
  }

  return (
    <SidePanelPage>
      <>
        <MainContent className={baseClass}>
          <>
            <div className={`${baseClass}__manage`}>
              <h1>My account</h1>
              <UserSettingsForm
                defaultData={{
                  email: currentUser.email,
                  name: currentUser.name,
                  position: currentUser.position,
                }}
                handleSubmit={handleSubmit}
                onCancel={onCancel}
                pendingEmail={pendingEmail}
                serverErrors={errors}
                smtpConfigured={config?.smtp_settings?.configured || false}
              />
            </div>
            {renderEmailModal()}
            {renderPasswordModal()}
            {renderApiTokenModal()}
          </>
        </MainContent>
        <SidePanelContent>
          <AccountSidePanel
            currentUser={currentUser}
            onChangePassword={onShowPasswordModal}
            onGetApiToken={onShowApiTokenModal}
          />
        </SidePanelContent>
      </>
    </SidePanelPage>
  );
};

export default AccountPage;
