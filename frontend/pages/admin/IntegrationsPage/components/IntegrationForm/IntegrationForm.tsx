import React, { FormEvent, useState, useEffect } from "react";
import { Link } from "react-router";
import PATHS from "router/paths";
import { useDispatch } from "react-redux";

import { ITeam } from "interfaces/team";
import { IUserFormErrors } from "interfaces/user"; // @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
// ignore TS error for now until these are rewritten in ts.
import Button from "components/buttons/Button";
import validatePresence from "components/forms/validators/validate_presence";
import validEmail from "components/forms/validators/valid_email"; // @ts-ignore
import validPassword from "components/forms/validators/valid_password"; // @ts-ignore
import InputField from "components/forms/fields/InputField";
import Checkbox from "components/forms/fields/Checkbox"; // @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import InfoBanner from "components/InfoBanner/InfoBanner";
import OpenNewTabIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";

const baseClass = "add-integration-form";

export enum NewUserType {
  AdminInvited = "ADMIN_INVITED",
  AdminCreated = "ADMIN_CREATED",
}

enum UserTeamType {
  GlobalUser = "GLOBAL_USER",
  AssignTeams = "ASSIGN_TEAMS",
}

const globalUserRoles = [
  {
    disabled: false,
    label: "Observer",
    value: "observer",
  },
  {
    disabled: false,
    label: "Maintainer",
    value: "maintainer",
  },
  {
    disabled: false,
    label: "Admin",
    value: "admin",
  },
];

export interface IFormData {
  email: string;
  name: string;
  newUserType?: NewUserType | null;
  password?: string | null;
  sso_enabled?: boolean;
  global_role: string | null;
  teams: ITeam[];
  currentUserId?: number;
  invited_by?: number;
}

interface ICreateUserFormProps {
  availableTeams: ITeam[];
  onCancel: () => void;
  onSubmit: (formData: IFormData) => void;
  submitText: string;
  defaultName?: string;
  defaultEmail?: string;
  currentUserId?: number;
  currentTeam?: ITeam;
  isModifiedByGlobalAdmin?: boolean | false;
  defaultGlobalRole?: string | null;
  defaultTeamRole?: string;
  defaultTeams?: ITeam[];
  isPremiumTier: boolean;
  smtpConfigured?: boolean;
  canUseSso: boolean; // corresponds to whether SSO is enabled for the organization
  isSsoEnabled?: boolean; // corresponds to whether SSO is enabled for the individual user
  isNewUser?: boolean;
  isInvitePending?: boolean;
  serverErrors?: { base: string; email: string }; // "server" because this form does its own client validation
  createOrEditUserErrors?: IUserFormErrors;
}

const IntegrationForm = ({
  onCancel,
  onSubmit,
  submitText,
  defaultName,
  defaultEmail,
  currentUserId,
  defaultGlobalRole,
  defaultTeams,
  isPremiumTier,
  smtpConfigured,
  isSsoEnabled,
  isNewUser,
  isInvitePending,
  serverErrors,
  createOrEditUserErrors,
}: ICreateUserFormProps): JSX.Element => {
  const dispatch = useDispatch();

  const [errors, setErrors] = useState<any>(createOrEditUserErrors);
  const [formData, setFormData] = useState<any>({
    email: defaultEmail || "",
    name: defaultName || "",
    newUserType: isNewUser ? NewUserType.AdminCreated : null,
    password: null,
    sso_enabled: isSsoEnabled,
    global_role: defaultGlobalRole || null,
    teams: defaultTeams || [],
    currentUserId,
  });

  useEffect(() => {
    setErrors(createOrEditUserErrors);
  }, [createOrEditUserErrors]);

  const onInputChange = (formField: string): ((value: string) => void) => {
    return (value: string) => {
      setErrors({
        ...errors,
        [formField]: null,
      });
      setFormData({
        ...formData,
        [formField]: value,
      });
    };
  };

  // UserForm component can be used to create a new user or edit an existing user so submitData will be assembled accordingly
  const createSubmitData = (): IFormData => {
    const submitData = formData;

    if (!isNewUser && !isInvitePending) {
      // if a new password is being set for an existing user, the API expects `new_password` rather than `password`
      submitData.new_password = formData.password;
      delete submitData.password;
      delete submitData.newUserType; // this field will not be submitted when form is used to edit an existing user
    }

    if (
      submitData.sso_enabled ||
      formData.newUserType === NewUserType.AdminInvited
    ) {
      delete submitData.password; // this field will not be submitted with the form
    }

    return { ...submitData, global_role: null, teams: formData.teams };
  };

  const validate = (): boolean => {
    if (!validatePresence(formData.email)) {
      setErrors({
        ...errors,
        email: "Email field must be completed",
      });

      return false;
    }

    if (!validEmail(formData.email)) {
      setErrors({
        ...errors,
        email: `${formData.email} is not a valid email`,
      });

      return false;
    }

    if (
      !isNewUser &&
      !isInvitePending &&
      formData.password &&
      !validPassword(formData.password)
    ) {
      setErrors({
        ...errors,
        password: "Password must meet the criteria below",
      });

      return false;
    }

    if (
      isNewUser &&
      formData.newUserType === NewUserType.AdminCreated &&
      !formData.sso_enabled
    ) {
      if (!validatePresence(formData.password)) {
        setErrors({
          ...errors,
          password: "Password field must be completed",
        });

        return false;
      }
      if (!validPassword(formData.password)) {
        setErrors({
          ...errors,
          password: "Password must meet the criteria below",
        });

        return false;
      }
    }

    if (!formData.global_role && !formData.teams.length) {
      dispatch(
        renderFlash("error", `Please select at least one team for this user.`)
      );
      return false;
    }

    return true;
  };

  const onFormSubmit = (evt: FormEvent): void => {
    evt.preventDefault();
    const valid = validate();
    if (valid) {
      return onSubmit(createSubmitData());
    }
  };

  return (
    <form className={baseClass} autoComplete="off">
      <InputField
        label="Full name"
        autofocus
        error={errors.name}
        name="name"
        onChange={onInputChange("name")}
        placeholder="Full name"
        value={formData.name || ""}
        inputOptions={{
          maxLength: "80",
        }}
      />
      <InputField
        label="Email"
        error={errors.email || serverErrors?.email}
        name="email"
        onChange={onInputChange("email")}
        placeholder="Email"
        value={formData.email || ""}
        disabled={!isNewUser && !smtpConfigured}
        tooltip={
          "\
              Editing an email address requires that SMTP is configured in order to send a validation email. \
              <br /><br /> \
              Users with Admin role can configure SMTP in <strong>Settings &gt; Organization settings</strong>. \
            "
        }
      />

      <div className={`${baseClass}__btn-wrap`}>
        <Button
          className={`${baseClass}__btn`}
          type="submit"
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
};

export default IntegrationForm;
