import React from "react";
import { SingleValue } from "react-select-5";

import Button from "components/buttons/Button";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import InputField from "components/forms/fields/InputField";
import TooltipWrapper from "components/TooltipWrapper";
import { IExternalIdentity } from "interfaces/external_identity";

const baseClass = "external-identities-section";

// user_external_identities.subject is VARCHAR(255).
export const EXTERNAL_IDENTITY_SUBJECT_MAX_LENGTH = 255;

export const externalIdentityErrorKey = (index: number) =>
  `external_identities.${index}.subject`;

interface IExternalIdentitiesSectionProps {
  identities: IExternalIdentity[];
  /** Names of the configured `auth.jwt_issuers`. */
  issuers: string[];
  onChange: (identities: IExternalIdentity[]) => void;
  getError: (name: string) => string | undefined;
  onSubjectFocus: (index: number) => void;
  disabled?: boolean;
}

const ExternalIdentitiesSection = ({
  identities,
  issuers,
  onChange,
  getError,
  onSubjectFocus,
  disabled,
}: IExternalIdentitiesSectionProps) => {
  const issuerOptions: CustomOptionType[] = issuers.map((name) => ({
    label: name,
    value: name,
  }));

  const update = (index: number, changes: Partial<IExternalIdentity>) =>
    onChange(
      identities.map((identity, i) =>
        i === index ? { ...identity, ...changes } : identity
      )
    );

  return (
    <div className={`${baseClass} form-field`}>
      <div className="form-field__label">
        <TooltipWrapper tipContent="A JWT from one of these identities authenticates as this user, so automations don't need a stored API token.">
          External identities
        </TooltipWrapper>
      </div>
      {identities.map((identity, index) => (
        // Rows have no stable ID until saved, and reordering isn't possible.
        // eslint-disable-next-line react/no-array-index-key
        <div className={`${baseClass}__row`} key={index}>
          <DropdownWrapper
            name={`external-identity-issuer-${index}`}
            label="Issuer"
            wrapperClassname={`${baseClass}__issuer`}
            value={identity.issuer}
            options={issuerOptions}
            onChange={(option: SingleValue<CustomOptionType>) =>
              option && update(index, { issuer: option.value })
            }
            isSearchable={false}
            isDisabled={disabled}
          />
          <InputField
            name={`external-identity-subject-${index}`}
            label="Subject"
            inputWrapperClass={`${baseClass}__subject`}
            value={identity.subject}
            onChange={(subject: string) => update(index, { subject })}
            onFocus={() => onSubjectFocus(index)}
            error={getError(externalIdentityErrorKey(index))}
            placeholder="repo:my-org/fleet-gitops:environment:production"
            disabled={disabled}
            inputOptions={{ maxLength: EXTERNAL_IDENTITY_SUBJECT_MAX_LENGTH }}
          />
          <Button
            variant="subdued"
            icon="trash"
            ariaLabel={`Remove external identity ${index + 1}`}
            onClick={() => onChange(identities.filter((_, i) => i !== index))}
            disabled={disabled}
          />
        </div>
      ))}
      <Button
        variant="subdued"
        icon="plus"
        onClick={() =>
          onChange([...identities, { issuer: issuers[0], subject: "" }])
        }
        disabled={disabled}
      >
        Add external identity
      </Button>
    </div>
  );
};

export default ExternalIdentitiesSection;
