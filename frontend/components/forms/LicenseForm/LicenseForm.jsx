import React, { Component, PropTypes } from 'react';

import Button from 'components/buttons/Button';
import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import InputField from 'components/forms/fields/InputField';
import validate from 'components/forms/LicenseForm/validate';

import freeTrial from '../../../../assets/images/sign-up-pencil.svg';
import key from '../../../../assets/images/key.svg';

const fields = ['license'];
const baseClass = 'license-form';

class LicenseForm extends Component {
  static propTypes = {
    baseError: PropTypes.string,
    fields: PropTypes.shape({
      license: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
  };

  render () {
    const { baseError, fields: formFields, handleSubmit } = this.props;

    return (
      <form className={baseClass} onSubmit={handleSubmit}>
        <div className={`${baseClass}__container`}>
          <h2>
            <img alt="Kolide License" className={`${baseClass}__key-img`} src={key} />
            Kolide License
          </h2>
          {baseError && <div className="form__base-error">{baseError}</div>}
          <InputField
            {...formFields.license}
            hint={<p className={`${baseClass}__help-text`}>Found under <a href="https://www.kolide.co/account">Account Settings</a> at Kolide.co</p>}
            inputClassName={`${baseClass}__input`}
            label="Enter License File"
            type="textarea"
          />
          <Button block className={`${baseClass}__upload-btn`} type="submit">
            UPLOAD LICENSE
          </Button>
          <p className="form-field__label">Don&apos;t have a license?</p>
          <p className={`${baseClass}__free-trial-text`}>Start a free trial of Kolide today!</p>
          <a
            className={`${baseClass}__free-trial-btn button button--unstyled`}
            href="https://www.kolide.co/register"
          >
            <img
              alt="Free trial"
              src={freeTrial}
              className={`${baseClass}__free-trial-img`}
            />
            <span>Sign up for Free Kolide Trial</span>
          </a>
        </div>
      </form>
    );
  }
}

export default Form(LicenseForm, { fields, validate });
