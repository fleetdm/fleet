import React, { Component, FormEvent } from 'react';

import { IUser } from 'interfaces/user';
import Button from 'components/buttons/Button';
import validatePresence from 'components/forms/validators/validate_presence';
import validEmail from 'components/forms/validators/valid_email';

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import InputFieldWithIcon from 'components/forms/fields/InputFieldWithIcon';
// @ts-ignore
import Checkbox from 'components/forms/fields/Checkbox';
import SelectedTeamsForm from '../SelectedTeamsForm/SelectedTeamsForm';

const baseClass = 'create-user-form';

interface IFormData {
  admin: boolean;
  email: string;
  name: string;
  sso_enabled: boolean;
  invited_by?: number;
}

interface ISubmitData extends IFormData {
  created_by: number
}

interface ICreateUserFormProps {
  createdBy: IUser;
  onCancel: () => void;
  onSubmit: (formData: ISubmitData) => void;
  canUseSSO: boolean;
}

interface ICreateUserFormState {
  errors: {
    admin: boolean | null;
    email: string | null;
    name: string | null;
    sso_enabled: boolean | null;
  };
  formData: IFormData
}

class CreateUserForm extends Component <ICreateUserFormProps, ICreateUserFormState> {
  constructor (props: ICreateUserFormProps) {
    super(props);

    this.state = {
      errors: {
        admin: null,
        email: null,
        name: null,
        sso_enabled: null,
      },
      formData: {
        admin: false,
        email: '',
        name: '',
        sso_enabled: false,
      },
    };
  }

  onInputChange = (formField: string) => {
    return (value: string) => {
      const { errors, formData } = this.state;

      this.setState({
        errors: {
          ...errors,
          [formField]: null,
        },
        formData: {
          ...formData,
          [formField]: value,
        },
      });
    };
  }

  onCheckboxChange = (formField: string) => {
    return (evt: string) => {
      return this.onInputChange(formField)(evt);
    };
  };

  onFormSubmit = (evt: FormEvent) => {
    evt.preventDefault();
    const valid = this.validate();

    if (valid) {
      const { formData: { admin, email, name, sso_enabled } } = this.state;
      const { createdBy, onSubmit } = this.props;
      return onSubmit({
        admin,
        email,
        created_by: createdBy.id,
        name,
        sso_enabled,
      });
    }
  }

  validate = (): boolean => {
    const {
      errors,
      formData: { email },
    } = this.state;

    if (!validatePresence(email)) {
      this.setState({
        errors: {
          ...errors,
          email: 'Email field must be completed',
        },
      });

      return false;
    }

    if (!validEmail(email)) {
      this.setState({
        errors: {
          ...errors,
          email: `${email} is not a valid email`,
        },
      });

      return false;
    }

    return true;
  }

  render () {
    const { errors, formData: { admin, email, name, sso_enabled } } = this.state;
    const { onCancel } = this.props;
    const { onFormSubmit, onInputChange, onCheckboxChange } = this;

    return (
      <form onSubmit={onFormSubmit} className={baseClass}>
        {/* {baseError && <div className="form__base-error">{baseError}</div>} */}
        <InputFieldWithIcon
          autofocus
          error={errors.name}
          name="name"
          onChange={onInputChange('name')}
          placeholder="Full Name"
          value={name}
        />
        <InputFieldWithIcon
          error={errors.email}
          name="email"
          onChange={onInputChange('email')}
          placeholder="Email"
          value={email}
        />
        <div className={`${baseClass}__radio`}>
          <div className={`${baseClass}__radio`}>
            <Checkbox
              name="sso_enabled"
              onChange={onCheckboxChange('sso_enabled')}
              value={sso_enabled}
              disabled={!this.props.canUseSSO}
              wrapperClassName={`${baseClass}__invite-admin`}
            >
              Enable Single Sign On
            </Checkbox>
          </div>

          <p className={`${baseClass}__role`}>Admin</p>
          <Checkbox
            name="admin"
            onChange={onCheckboxChange('admin')}
            value={admin}
            wrapperClassName={`${baseClass}__invite-admin`}
          >
            Enable Admin
          </Checkbox>
        </div>

        <div className={`${baseClass}__selected-teams-container`}>
          <SelectedTeamsForm
            teams={[{ name: 'Test Team', id: 1, role: 'admin' }, { name: 'Test Team 2', id: 2, role: 'admin' }]}
          />
        </div>

        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={() => { return null; }}
          >
            Create
          </Button>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="inverse"
          >
            Cancel
          </Button>
        </div>
      </form>
    );
  }
}

export default CreateUserForm;
