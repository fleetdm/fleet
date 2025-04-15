import React, { useEffect } from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import { IRegistrationFormData } from "interfaces/registration_form_data";
import Checkbox from "components/forms/fields/Checkbox";

const baseClass = "confirm-user-reg";

interface IConfirmationPageProps {
  className: string;
  currentPage: boolean;
  formData: IRegistrationFormData;
  handleSubmit: React.FormEventHandler<HTMLFormElement>;
}

const ConfirmationPage = ({
  className,
  currentPage,
  formData,
  handleSubmit,
}: IConfirmationPageProps): JSX.Element => {
  useEffect(() => {
    if (currentPage) {
      // Component has a transition duration of 300ms set in
      // RegistrationForm/_styles.scss. We need to wait 300ms before
      // calling .focus() to preserve smooth transition.
      setTimeout(() => {
        // wanted to use React ref here instead of class but ref is already used
        // in Button.tsx, which could break other button uses
        const confirmationButton = document.querySelector(
          `.${baseClass} button.button--default`
        ) as HTMLElement;
        confirmationButton?.focus();
      }, 300);
    }
  }, [currentPage]);

  const importOsqueryConfig = () => {
    const disableImport = true;

    if (disableImport) {
      return false;
    }

    return (
      <div className={`${baseClass}__import`}>
        <Checkbox name="import-install">
          <p>
            I am migrating an existing <strong>osquery</strong> installation.
          </p>
          <p>
            Take me to the <strong>Import Configuration</strong> page.
          </p>
        </Checkbox>
      </div>
    );
  };

  const { email, server_url: serverUrl, org_name: orgName, name } = formData;
  const tabIndex = currentPage ? 0 : -1;

  const confirmRegClasses = classnames(className, baseClass);

  return (
    <form
      onSubmit={handleSubmit}
      className={confirmRegClasses}
      autoComplete="off"
    >
      <div className={`${baseClass}__wrapper`}>
        <table className={`${baseClass}__table`}>
          <caption>Administrator configuration</caption>
          <tbody>
            <tr>
              <th>Full name:</th>
              <td>{name}</td>
            </tr>
            <tr>
              <th>Email:</th>
              <td className={`${baseClass}__table-email`}>{email}</td>
            </tr>
            <tr>
              <th>Organization:</th>
              <td>{orgName}</td>
            </tr>
            <tr>
              <th>Fleet URL:</th>
              <td>
                <span className={`${baseClass}__table-url`} title={serverUrl}>
                  {serverUrl}
                </span>
              </td>
            </tr>
          </tbody>
        </table>

        {importOsqueryConfig()}
      </div>
      <p className="help-text">
        Fleet Device Management Inc. periodically collects information about
        your instance. Sending usage statistics from your Fleet instance is
        optional and can be disabled in settings.
      </p>
      <Button type="submit" tabIndex={tabIndex} disabled={!currentPage}>
        Confirm
      </Button>
    </form>
  );
};

export default ConfirmationPage;
