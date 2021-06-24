import React, { Component } from "react";
import PropTypes from "prop-types";
import { Link } from "react-router";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import paths from "router/paths";
import validate from "components/forms/LoginForm/validate";
import ssoSettingsInterface from "interfaces/ssoSettings";

const baseClass = "login-form";
const formFields = ["email", "password"];

class LoginForm extends Component {
  static propTypes = {
    baseError: PropTypes.string,
    fields: PropTypes.shape({
      password: formFieldInterface.isRequired,
      email: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func,
    isHidden: PropTypes.bool,
    ssoSettings: ssoSettingsInterface,
    handleSSOSignOn: PropTypes.func,
  };

  showLegendWithImage = (image, idpName) => {
    let legend = "Single Sign On";
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

  showSingleSignOnButton = () => {
    const { ssoSettings, handleSSOSignOn } = this.props;
    const { idp_name: idpName, idp_image_url: imageURL } = ssoSettings;
    const { showLegendWithImage } = this;

    let legend = "Single Sign On";
    if (idpName !== "") {
      legend = `Sign On With ${idpName}`;
    }
    if (imageURL !== "") {
      legend = showLegendWithImage(imageURL, idpName);
    }

    return (
      <Button
        className={`${baseClass}__sso-btn`}
        type="button"
        title="Single Sign On"
        variant="inverse"
        onClick={handleSSOSignOn}
      >
        <div>{legend}</div>
      </Button>
    );
  };

  render() {
    const {
      baseError,
      fields,
      handleSubmit,
      isHidden,
      ssoSettings,
    } = this.props;
    const { sso_enabled: ssoEnabled } = ssoSettings;
    const { showSingleSignOnButton } = this;

    const loginFormClass = classnames(baseClass, {
      [`${baseClass}--hidden`]: isHidden,
    });

    return (
      <form onSubmit={handleSubmit} className={loginFormClass}>
        <div className={`${baseClass}__container`}>
          {baseError && <div className="form__base-error">{baseError}</div>}
          <InputFieldWithIcon {...fields.email} autofocus placeholder="Email" />
          <InputFieldWithIcon
            {...fields.password}
            placeholder="Password"
            type="password"
          />
          <div className={`${baseClass}__forgot-wrap`}>
            <Link
              className={`${baseClass}__forgot-link`}
              to={paths.FORGOT_PASSWORD}
            >
              Forgot Password?
            </Link>
          </div>
          <Button
            className={`${baseClass}__submit-btn button button--brand`}
            onClick={handleSubmit}
            type="submit"
          >
            Login
          </Button>
          {ssoEnabled && showSingleSignOnButton()}
        </div>
      </form>
    );
  }
}

export default Form(LoginForm, {
  fields: formFields,
  validate,
});
