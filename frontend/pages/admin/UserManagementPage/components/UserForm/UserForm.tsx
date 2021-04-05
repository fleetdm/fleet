import React, { Component, FormEvent } from 'react';

import { ITeam } from 'interfaces/team';
import Button from 'components/buttons/Button';
import validatePresence from 'components/forms/validators/validate_presence';
import validEmail from 'components/forms/validators/valid_email';

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import InputFieldWithIcon from 'components/forms/fields/InputFieldWithIcon';
// @ts-ignore
import Checkbox from 'components/forms/fields/Checkbox';
// @ts-ignore
import Dropdown from 'components/forms/fields/Dropdown';
import Radio from 'components/forms/fields/Radio';
import InfoBanner from 'components/InfoBanner/InfoBanner';
import SelectedTeamsForm from '../SelectedTeamsForm/SelectedTeamsForm';
import OpenNewTabIcon from '../../../../../../assets/images/open-new-tab-12x12@2x.png';

const baseClass = 'create-user-form';

enum UserTeamType {
  GlobalUser = 'GLOBAL_USER',
  AssignTeams = 'ASSIGN_TEAMS',
}

const globalUserRoles = [
  {
    disabled: false,
    label: 'observer',
    value: 'observer',
  },
  {
    disabled: false,
    label: 'maintainer',
    value: 'maintainer',
  },
  {
    disabled: false,
    label: 'admin',
    value: 'admin',
  },
];

interface IFormData {
  email: string;
  name: string;
  sso_enabled: boolean;
  currentUserId: number;
  global_role: string | null;
  teams: ITeam[];
  invited_by?: number;
}

interface ICreateUserFormProps {
  availableTeams: ITeam[];
  currentUserId: number;
  onCancel: () => void;
  onSubmit: (formData: IFormData) => void;
  canUseSSO: boolean;
  submitText: string;
  defaultName?: string;
  defaultEmail?: string;
  defaultGlobalRole?: string | null;
  defaultTeams: ITeam[];
}

interface ICreateUserFormState {
  errors: {
    email: string | null;
    name: string | null;
    sso_enabled: boolean | null;
  };
  formData: IFormData,
  isGlobalUser: boolean,
}

class UserForm extends Component <ICreateUserFormProps, ICreateUserFormState> {
  constructor (props: ICreateUserFormProps) {
    super(props);

    this.state = {
      errors: {
        email: null,
        name: null,
        sso_enabled: null,
      },
      formData: {
        email: props.defaultEmail || '',
        name: props.defaultName || '',
        sso_enabled: false,
        global_role: props.defaultGlobalRole || null,
        teams: props.defaultTeams,
        currentUserId: props.currentUserId,
      },
      isGlobalUser: props.defaultGlobalRole !== null,
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

  onIsGlobalUserChange = (value: string): void => {
    const isGlobalUser = value === UserTeamType.GlobalUser;
    this.setState({
      isGlobalUser,
    });
  }

  onGlobalUserRoleChange = (value: string): void => {
    const { formData } = this.state;
    this.setState({
      formData: {
        ...formData,
        global_role: value,
      },
    });
  }

  onSelectedTeamChange = (teams: ITeam[]): void => {
    const { formData } = this.state;
    this.setState({
      formData: {
        ...formData,
        teams,
      },
    });
  }

  onFormSubmit = (evt: FormEvent): void => {
    evt.preventDefault();
    const valid = this.validate();
    if (valid) {
      const { formData: { email, name, sso_enabled, global_role, teams } } = this.state;
      const { onSubmit, currentUserId } = this.props;
      return onSubmit({
        email,
        currentUserId,
        name,
        sso_enabled,
        global_role,
        teams,
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

  renderGlobalRoleForm = () => {
    const { onGlobalUserRoleChange } = this;
    const { formData: { global_role } } = this.state;
    return (
      <>
        <InfoBanner className={`${baseClass}__user-permissions-info`}>
          <p>Global users can only be members of the top level team and can manage or observe all users, entities, and settings in Fleet.</p>
          <a
            href="https://github.com/fleetdm/fleet/blob/master/docs/1-Using-Fleet/2-fleetctl-CLI.md#osquery-configuration-options"
            target="_blank"
            rel="noreferrer"
          >
            Learn more about user permissions
            <img src={OpenNewTabIcon} alt="open new tab" />
          </a>
        </InfoBanner>
        <p className={`${baseClass}__label`}>Role</p>
        <Dropdown
          value={global_role || 'observer'}
          className={`${baseClass}__global-role-dropdown`}
          options={globalUserRoles}
          searchable={false}
          onChange={onGlobalUserRoleChange}
        />
      </>
    );
  }


  renderTeamsForm = (): JSX.Element => {
    const { onSelectedTeamChange } = this;
    const { availableTeams } = this.props;
    const { formData: { teams } } = this.state;
    return (
      <>
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
          // availableTeams={availableTeams}
          availableTeams={[{ name: 'Test Team', id: 1, role: 'admin' }, { name: 'Test Team 2', id: 2, role: 'admin' }]}
          usersCurrentTeams={teams}
          onFormChange={onSelectedTeamChange}
        />
      </>
    );
  }

  render (): JSX.Element {
    const { errors, formData: { email, name, sso_enabled }, isGlobalUser } = this.state;
    const { onCancel, submitText } = this.props;
    const { onFormSubmit, onInputChange, onCheckboxChange, onIsGlobalUserChange, renderGlobalRoleForm, renderTeamsForm } = this;

    return (
      <form className={baseClass}>
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
        <div className={`${baseClass}__sso-input`}>
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

        <div className={`${baseClass}__selected-teams-container`}>
          <div className={`${baseClass}__team-radios`}>
            <p className={`${baseClass}__label`}>Team</p>
            <Radio
              className={`${baseClass}__radio-input`}
              label={'Global user'}
              id={'global-user'}
              checked={isGlobalUser}
              value={UserTeamType.GlobalUser}
              name={'userTeamType'}
              onChange={onIsGlobalUserChange}
            />
            <Radio
              className={`${baseClass}__radio-input`}
              label={'Assign teams'}
              id={'assign-teams'}
              checked={!isGlobalUser}
              value={UserTeamType.AssignTeams}
              name={'userTeamType'}
              onChange={onIsGlobalUserChange}
            />
          </div>
          <div className={`${baseClass}__teams-form-container`}>
            {isGlobalUser ? renderGlobalRoleForm() : renderTeamsForm()}
          </div>
        </div>

        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={onFormSubmit}
          >
            {submitText}
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

export default UserForm;
