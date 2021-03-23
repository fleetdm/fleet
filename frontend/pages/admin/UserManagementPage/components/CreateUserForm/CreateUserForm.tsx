import React, { Component, FormEvent } from 'react';

import { IUser } from 'interfaces/user';
import ITeam from 'interfaces/team';
import Button from 'components/buttons/Button';
import validatePresence from 'components/forms/validators/validate_presence';
import validEmail from 'components/forms/validators/valid_email';

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import InputFieldWithIcon from 'components/forms/fields/InputFieldWithIcon';
// @ts-ignore
import Checkbox from 'components/forms/fields/Checkbox';
import Radio from 'components/forms/fields/Radio';
import InfoBanner from 'components/InfoBanner/InfoBanner';
import SelectedTeamsForm from '../SelectedTeamsForm/SelectedTeamsForm';
import OpenNewTabIcon from '../../../../../../assets/images/open-new-tab-12x12@2x.png';

const baseClass = 'create-user-form';

interface IFormData {
  admin: boolean;
  email: string;
  name: string;
  sso_enabled: boolean;
  global_role?: string;
  selectedTeams?: ITeam[];
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
  availableTeams: ITeam[];
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
        global_role: '',
        selectedTeams: [],
      },
    };
  }

  onInputChange = (formField: string): (value: string) => void => {
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

  onCheckboxChange = (formField: string): (evt: string) => void => {
    return (evt: string) => {
      return this.onInputChange(formField)(evt);
    };
  };

  onSelectedTeamChange = (teams: ITeam[]): void => {
    const { formData } = this.state;
    this.setState({
      formData: {
        ...formData,
        selectedTeams: teams,
      },
    });
  }

  onFormSubmit = (evt: FormEvent): void => {
    evt.preventDefault();
    const valid = this.validate();
    if (valid) {
      const { formData: { admin, email, name, sso_enabled, global_role, selectedTeams } } = this.state;
      const { createdBy, onSubmit } = this.props;
      return onSubmit({
        admin,
        email,
        created_by: createdBy.id,
        name,
        sso_enabled,
        global_role,
        selectedTeams,
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

  render (): JSX.Element {
    const { errors, formData: { admin, email, name, sso_enabled } } = this.state;
    const { onCancel, availableTeams } = this.props;
    const { onFormSubmit, onInputChange, onCheckboxChange, onSelectedTeamChange } = this;

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
          <div className={`${baseClass}__team-radios`}>
            <Radio
              label={'Global user'}
              id={'global-user'}
              checked
              value={'globalUser'}
              onChange={value => console.log(value)}
            />
            <Radio
              label={'Assign teams'}
              id={'assign-teams'}
              value={'assignTeams'}
              onChange={value => console.log(value)}
            />
          </div>

          <InfoBanner className={`${baseClass}__user-permissions-info`}>
            <p>Users can be members of multiple teams and can only manage or observe team-sepcific users, entities, and settings in Fleet.</p>
            <a
              href="https://github.com/fleetdm/fleet/blob/master/docs/1-Using-Fleet/2-fleetctl-CLI.md#osquery-configuration-options"
              target="_blank"
              rel="noreferrer"
            >
              Learn more about user permissions
              <img src={OpenNewTabIcon} alt="open new tab" />
            </a>
          </InfoBanner>
          <SelectedTeamsForm
            availableTeams={[{ name: 'Test Team', id: 1, role: 'admin' }, { name: 'Test Team 2', id: 2, role: 'admin' }]}
            usersCurrentTeams={[]}
            onFormChange={onSelectedTeamChange}
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
