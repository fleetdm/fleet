import React, { useEffect, useState } from "react";
import classnames from "classnames";
import { ILoginUserData } from "interfaces/user";

import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import paths from "router/paths";
import { ISSOSettings } from "interfaces/ssoSettings";
import validatePresence from "components/forms/validators/validate_presence";
import validateEmail from "components/forms/validators/valid_email";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";

const baseClass = "login-form";

interface ILoginFormProps {
  baseError?: string;
  handleSubmit: (formData: ILoginUserData) => Promise<false | void>;
  isSubmitting: boolean;
  pendingEmail: boolean;
  ssoSettings?: ISSOSettings;
  handleSSOSignOn?: () => void;
}

const validate = ({ email, password }: ILoginUserData): IFormErrors => {
  const errors: IFormErrors = {};

  if (!validatePresence(email)) {
    errors.email = "Enter your email";
  } else if (!validateEmail(email)) {
    errors.email = "Enter a valid email";
  }

  if (!validatePresence(password)) {
    errors.password = "Enter your password";
  }

  return errors;
};

const LoginForm = ({
  baseError,
  handleSubmit,
  isSubmitting: isSubmittingExternal,
  pendingEmail,
  ssoSettings,
  handleSSOSignOn,
}: ILoginFormProps): JSX.Element => {
  const {
    idp_name: idpName,
    idp_image_url: imageURL,
    sso_enabled: ssoEnabled,
  } = ssoSettings || {}; // TODO: Consider refactoring ssoSettings undefined

  const loginFormClass = classnames(baseClass);

  const {
    formData,
    setField,
    getError,
    clearFieldError,
    validateField,
    handleSubmit: onSubmit,
    isSubmitting,
  } = useFormValidation<ILoginUserData>({
    initialFormData: { email: "", password: "" },
    validate,
    isSubmitting: isSubmittingExternal,
    skipTrim: ["password"],
  });

  const [showPendingEmail, setShowPendingEmail] = useState(pendingEmail);

  useEffect(() => {
    setShowPendingEmail(pendingEmail);
  }, [pendingEmail]);

  const renderSingleSignOnButton = () => {
    const button = (
      <Button
        className={`${baseClass}__sso-btn`}
        type="button"
        variant="secondary"
        onClick={handleSSOSignOn}
        tabIndex={0}
      >
        {imageURL && (
          <img src={imageURL} alt="" className={`${baseClass}__sso-image`} />
        )}
        <span className={`${baseClass}__sso-legend`}>Sign in with SSO</span>
      </Button>
    );

    // The label is always the generic "Sign in with SSO"; the configured IdP's
    // name is surfaced only on hover so a long name can't overflow the button.
    if (!idpName) {
      return button;
    }

    return (
      <TooltipWrapper
        className={`${baseClass}__sso-tooltip`}
        tipContent={`Sign in with ${idpName}`}
        position="top"
        showArrow
        underline={false}
      >
        {button}
      </TooltipWrapper>
    );
  };

  if (showPendingEmail) {
    return (
      <div className="two-factor-check-email">
        <>
          <Button
            onClick={() => setShowPendingEmail(false)}
            variant="subdued"
            className="back-link"
            icon="chevron-left"
          >
            Back to login
          </Button>
          <h1>Check your email</h1>
          <p className={`${baseClass}__text`}>
            We sent an email to you at <b>{formData.email}</b>. <br />
            Please click the magic link in the email to sign in.
          </p>
        </>
      </div>
    );
  }

  return (
    <form
      onSubmit={onSubmit(handleSubmit)}
      className={loginFormClass}
      noValidate
    >
      {baseError && <div className="form__base-error">{baseError}</div>}
      <div className={`${baseClass}__form`}>
        <InputFieldWithIcon
          error={getError("email")}
          autofocus
          type="email"
          label="Email"
          placeholder="Email"
          value={formData.email}
          onChange={(value: string) => setField("email", value)}
          onFocus={() => clearFieldError("email")}
          onBlur={() => validateField("email")}
          ignore1Password={false}
          disabled={isSubmitting}
        />
        <InputFieldWithIcon
          error={getError("password")}
          label="Password"
          placeholder="Password"
          type="password"
          value={formData.password}
          onChange={(value: string) => setField("password", value)}
          onFocus={() => clearFieldError("password")}
          onBlur={() => validateField("password")}
          ignore1Password={false}
          disabled={isSubmitting}
        />
      </div>
      {/* Actions displayed using CSS column-reverse to preserve tab order */}
      <div className={`${baseClass}__actions`}>
        <div className={`${baseClass}__login-actions`}>
          <Button
            className={`${baseClass}__login-btn`}
            isLoading={isSubmitting}
            disabled={isSubmitting}
            type="submit"
            tabIndex={0}
          >
            Log in
          </Button>
          {ssoEnabled && renderSingleSignOnButton()}
        </div>
        <CustomLink
          className={`${baseClass}__forgot-link`}
          url={paths.FORGOT_PASSWORD}
          text="Forgot password?"
        />
      </div>
    </form>
  );
};

export default LoginForm;
