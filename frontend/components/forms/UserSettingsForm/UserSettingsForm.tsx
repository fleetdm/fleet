import React from "react";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import { isPresent, isValidEmail } from "components/forms/validators";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";

const baseClass = "manage-user";

// `users.name`, `users.email`, and `users.position` are all varchar(255).
const USER_TEXT_MAX_LENGTH = 255;

export interface IUserSettingsFormData {
  email: string;
  name: string;
  position?: string;
}

interface IUserSettingsFormProps {
  defaultData?: Partial<IUserSettingsFormData>;
  handleSubmit: (formData: IUserSettingsFormData) => void | Promise<unknown>;
  onCancel: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  pendingEmail?: string;
  serverErrors?: IFormErrors | null;
  smtpConfigured?: boolean;
}

const UserSettingsForm = ({
  defaultData,
  handleSubmit,
  onCancel,
  pendingEmail,
  serverErrors,
  smtpConfigured,
}: IUserSettingsFormProps): JSX.Element => {
  const isEmailReadOnly = !smtpConfigured;

  const validate = (data: IUserSettingsFormData): IFormErrors => {
    const errors: IFormErrors = {};

    if (!isEmailReadOnly) {
      if (!isPresent(data.email)) {
        errors.email = "Enter your email";
      } else if (!isValidEmail(data.email)) {
        errors.email = "Enter a valid email";
      }
    }

    if (!isPresent(data.name)) {
      errors.name = "Enter your name";
    }

    return errors;
  };

  const {
    getTextFieldProps,
    handleSubmit: onSubmit,
    isSubmitting,
  } = useFormValidation<IUserSettingsFormData>({
    initialFormData: {
      email: defaultData?.email ?? "",
      name: defaultData?.name ?? "",
      position: defaultData?.position,
    },
    validate,
    serverErrors,
  });

  const emailHelpText = pendingEmail ? (
    <i className={`${baseClass}__email-help-text`}>
      Pending change to <b>{pendingEmail}</b>
    </i>
  ) : undefined;

  return (
    <form
      onSubmit={onSubmit(handleSubmit)}
      className={baseClass}
      autoComplete="off"
    >
      <InputField
        {...getTextFieldProps("email")}
        autofocus
        label="Email"
        helpText={emailHelpText}
        readOnly={isEmailReadOnly}
        disabled={isSubmitting}
        ignore1password={false}
        inputOptions={{ maxLength: USER_TEXT_MAX_LENGTH }}
        tooltip={
          <>
            Editing your email address requires that SMTP or SES is configured
            in order to send a validation email.
            <br />
            <br />
            Users with Admin role can configure SMTP in{" "}
            <strong>Settings &gt; Organization settings</strong>.
          </>
        }
      />
      <InputField
        {...getTextFieldProps("name")}
        label="Full name"
        disabled={isSubmitting}
        inputOptions={{ maxLength: USER_TEXT_MAX_LENGTH }}
      />
      <InputField
        {...getTextFieldProps("position")}
        label="Position"
        disabled={isSubmitting}
        inputOptions={{ maxLength: USER_TEXT_MAX_LENGTH }}
      />
      <div className="button-wrap">
        <Button onClick={onCancel} variant="secondary">
          Cancel
        </Button>
        <Button type="submit" isLoading={isSubmitting} disabled={isSubmitting}>
          Update
        </Button>
      </div>
    </form>
  );
};

export default UserSettingsForm;
