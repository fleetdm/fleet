import React, { FormEvent, useState } from "react";
import { Link } from "react-router";
import { size } from "lodash";
import classnames from "classnames";

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
  handleSubmit: () => void;
  isHidden: boolean;
  ssoSettings: ISSOSettings;
  handleSSOSignOn: () => void;
}

const LoginForm = ({
  baseError,
  handleSubmit,
  isHidden,
  ssoSettings,
  handleSSOSignOn,
}: ILoginFormProps): JSX.Element => {
  const [errors, setErrors] = useState<any>({}); // TODO
  const [formData, setFormData] = useState<any>({
    email: "",
    password: "",
  }); // TODO

  // TODO
  const validate = () => {
    const { password, email } = formData;

    if (!validatePresence(email)) {
      errors.email = "Email field must be completed";
    } else if (!validateEmail(email)) {
      errors.email = "Email must be a valid email address";
    }

    if (!validatePresence(password)) {
      errors.password = "Password field must be completed";
    }

    const valid = !size(errors);

    return { valid, errors };
  };

  const onFormSubmit = (evt: FormEvent): void => {
    evt.preventDefault();
    const valid = validate();
    if (valid) {
      return handleSubmit();
    }
  };

  const showLegendWithImage = (image: string, idpName: string) => {
    let legend = "Single sign-on";
    if (idpName !== "") {
      legend = `Sign on with ${idpName}`;
    }

    return (
      <div>
        <img src={image} alt={idpName} className={`${baseClass}__sso-image`} />
        <span className={`${baseClass}__sso-legend`}>{legend}</span>
      </div>
    );
  };

  const renderSingleSignOnButton = () => {
    const { idp_name: idpName, idp_image_url: imageURL } = ssoSettings;

    let legend: string | JSX.Element = "Single sign-on";
    if (idpName !== "") {
      legend = `Sign on with ${idpName}`;
    }
    if (imageURL !== "") {
      legend = showLegendWithImage(imageURL, idpName);
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

  const { sso_enabled: ssoEnabled } = ssoSettings || {};

  const loginFormClass = classnames(baseClass, {
    [`${baseClass}--hidden`]: isHidden,
  });

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
      <div className={`${baseClass}__container`}>
        {baseError && <div className="form__base-error">{baseError}</div>}
        <InputFieldWithIcon
          error={errors.email}
          autofocus
          label="Email"
          placeholder="Email"
          value={formData.email || ""}
          onChange={onInputChange("email")}
        />
        <InputFieldWithIcon
          error={errors.password}
          label="Password"
          placeholder="Password"
          type="password"
          value={formData.password || ""}
          onChange={onInputChange("password")}
        />
        <div className={`${baseClass}__forgot-wrap`}>
          <Link
            className={`${baseClass}__forgot-link`}
            to={paths.FORGOT_PASSWORD}
          >
            Forgot password?
          </Link>
        </div>
        <Button
          className={`${baseClass}__submit-btn button button--brand`}
          onClick={handleSubmit}
          type="submit"
        >
          Login
        </Button>
        {ssoEnabled && renderSingleSignOnButton()}
      </div>
    </form>
  );
};

export default LoginForm;
