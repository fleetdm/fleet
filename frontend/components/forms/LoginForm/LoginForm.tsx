import React, { FormEvent, useState } from "react";
import { Link } from "react-router";
import { size } from "lodash";
import classnames from "classnames";
import { ILoginUserData } from "interfaces/user";

import Button from "components/buttons/Button";
// @ts-ignore
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import paths from "router/paths";
import { ISSOSettings } from "interfaces/ssoSettings";
import validatePresence from "components/forms/validators/validate_presence";
import validateEmail from "components/forms/validators/valid_email";

const baseClass = "login-form";

interface ILoginFormProps {
  baseError?: string;
  handleSubmit: (formData: ILoginUserData) => Promise<false | void>;
  ssoSettings?: ISSOSettings;
  handleSSOSignOn?: () => void;
}

const LoginForm = ({
  baseError,
  handleSubmit,
  ssoSettings,
  handleSSOSignOn,
}: ILoginFormProps): JSX.Element => {
  const {
    idp_name: idpName,
    idp_image_url: imageURL,
    sso_enabled: ssoEnabled,
  } = ssoSettings || {}; // TODO: Consider refactoring ssoSettings undefined

  const loginFormClass = classnames(baseClass);

  const [errors, setErrors] = useState<any>({});
  const [formData, setFormData] = useState<ILoginUserData>({
    email: "",
    password: "",
  });

  const validate = () => {
    const { password, email } = formData;

    const validationErrors: { [key: string]: string } = {};

    if (!validatePresence(email)) {
      validationErrors.email = "Email field must be completed";
    } else if (!validateEmail(email)) {
      validationErrors.email = "Email must be a valid email address";
    }

    if (!validatePresence(password)) {
      validationErrors.password = "Password field must be completed";
    }

    setErrors(validationErrors);
    const valid = !size(validationErrors);

    return valid;
  };

  const onFormSubmit = (evt: FormEvent): Promise<false | void> | boolean => {
    evt.preventDefault();
    const valid = validate();

    if (valid) {
      return handleSubmit(formData);
    }
    return false;
  };

  const showLegendWithImage = () => {
    let legend = "Single sign-on";
    if (idpName !== "") {
      legend = `Sign on with ${idpName}`;
    }

    return (
      <div>
        <img
          src={imageURL}
          alt={idpName}
          className={`${baseClass}__sso-image`}
        />
        <span className={`${baseClass}__sso-legend`}>{legend}</span>
      </div>
    );
  };

  const renderSingleSignOnButton = () => {
    let legend: string | JSX.Element = "Single sign-on";
    if (idpName !== "") {
      legend = `Sign on with ${idpName}`;
    }
    if (imageURL !== "") {
      legend = showLegendWithImage();
    }

    return (
      <Button
        className={`${baseClass}__sso-btn`}
        type="button"
        title="Single sign-on"
        variant="inverse"
        onClick={handleSSOSignOn}
      >
        <div>{legend}</div>
      </Button>
    );
  };

  const onInputChange = (formField: string): ((value: string) => void) => {
    return (value: string) => {
      setErrors({});
      setFormData({
        ...formData,
        [formField]: value,
      });
    };
  };

  return (
    <form onSubmit={onFormSubmit} className={loginFormClass}>
      {baseError && <div className="form__base-error">{baseError}</div>}
      <InputFieldWithIcon
        error={errors.email}
        autofocus
        label="Email"
        placeholder="Email"
        value={formData.email}
        onChange={onInputChange("email")}
      />
      <InputFieldWithIcon
        error={errors.password}
        label="Password"
        placeholder="Password"
        type="password"
        value={formData.password}
        onChange={onInputChange("password")}
      />
      {/* Actions displayed using CSS column-reverse to preserve tab order */}
      <div className={`${baseClass}__actions`}>
        <div className={`${baseClass}__login-actions`}>
          <Button className={`login-btn button button--brand`} type="submit">
            Log in
          </Button>
          {ssoEnabled && renderSingleSignOnButton()}
        </div>
        <div className={`${baseClass}__forgot-wrap`}>
          <Link
            className={`${baseClass}__forgot-link`}
            to={paths.FORGOT_PASSWORD}
          >
            Forgot password?
          </Link>
        </div>
      </div>
    </form>
  );
};

export default LoginForm;
