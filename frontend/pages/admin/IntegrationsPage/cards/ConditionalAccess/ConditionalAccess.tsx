import React, { useContext, useEffect, useState } from "react";

import { size } from "lodash";

import paths from "router/paths";

import { NotificationContext } from "context/notification";

import conditionalAccessAPI from "services/entities/conditional_access";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import CustomLink from "components/CustomLink";
import SectionHeader from "components/SectionHeader";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import Button from "components/buttons/Button";
import { IFormField } from "interfaces/form_field";
import { AppContext } from "context/app";
import Spinner from "components/Spinner";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import InfoBanner from "components/InfoBanner";
import Icon from "components/Icon";
import TooltipTruncatedText from "components/TooltipTruncatedText";
// import { getErrorReason } from "interfaces/errors";

const baseClass = "conditional-access";

const msetid = "microsoft_entra_tenant_id";

// States –> UI phases:
// 	- not premium –> -1
// 	- no tenant id –> 0
// 	- tenant id & form submitted –> 1
// 	- tenant id & no consent –> 2
//   - tenant id & consent –> 3

interface IFormData {
  [msetid]: string;
}

interface IFormErrors {
  [msetid]?: string | null;
}

const validate = (formData: IFormData) => {
  const errs: IFormErrors = {};
  if (!formData[msetid]) {
    errs[msetid] = "Tenant ID must be present";
  }
  return errs;
};

const ConditionalAccess = () => {
  const { renderFlash } = useContext(NotificationContext);

  // UI phases:
  // 	0: form (valid, loading, error)
  // 	1: form submitted (aka, “continue in other tab”)
  // 	2: checking integration
  // 	3: integration confirmed
  const [phase, setPhase] = useState(0);

  const [formData, setFormData] = useState<IFormData>({
    // [msetid]: "12345",
    [msetid]: "",
  });
  const [formErrors, setFormErrors] = useState<IFormErrors>({});
  const [isUpdating, setIsUpdating] = useState(false);

  const { isPremiumTier, config } = useContext(AppContext);

  // watch curMsetid coming from `config`, populate initial form state once present
  const curMsetid = config?.conditional_access?.microsoft_entra_tenant_id;
  useEffect(() => {
    setFormData({
      [msetid]:
        curMsetid ||
        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    });
  }, [curMsetid]);

  if (!isPremiumTier) {
    return <PremiumFeatureMessage />;
  }

  const handleSubmit = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const errs = validate(formData);
    if (Object.keys(errs).length > 0) {
      setFormErrors(errs);
      return;
    }
    setIsUpdating(true);
    try {
      // await conditionalAccessAPI.triggerMicrosoftConditionalAccess(
      //   formData[msetid]
      // );
      await setTimeout(() => true, 3000);
      setIsUpdating(false);
      setPhase(1);
      // TODO:
      // open a new tab navigating to the authentication URL returned from the API
      // (https://login.microsoftonline.com/{tenant-id}/adminconsent?client_id={client-id})
    } catch (e) {
      // const message = getErrorReason(e);
      // renderFlash("error", message || "Failed to update settings");
      // TODO - coordinate with Lucas re what this error will contain
      renderFlash(
        "error",
        "Could not update conditional access integration settings."
      );
      setIsUpdating(false);
    }
  };

  const onInputChange = ({ name, value }: IFormField<string>) => {
    const newFormData = { ...formData, [name]: value };
    setFormData(newFormData);
    const newErrs = validate(newFormData);
    // only set errors that are updates of existing errors
    // new errors are only set onBlur or submit
    const errsToSet: Record<string, string> = {};
    Object.keys(formErrors).forEach((k) => {
      // @ts-ignore
      if (newErrs[k]) {
        // @ts-ignore
        errsToSet[k] = newErrs[k];
      }
    });
    setFormErrors(errsToSet);
  };

  const onInputBlur = () => {
    setFormErrors(validate(formData));
  };

  const renderForm = () => (
    <form onSubmit={handleSubmit} autoComplete="off">
      <InputField
        label="Microsoft Entra tenant ID"
        helpText={
          <>
            You can find this in your Microsoft Entra admin center.{" "}
            <CustomLink
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/microsoft-entra-setup`}
              text="Learn more"
              newTab
            />
          </>
        }
        onChange={onInputChange}
        name={msetid}
        value={formData[msetid]}
        parseTarget
        onBlur={onInputBlur}
        error={formErrors[msetid]}
      />
      <Button
        type="submit"
        variant="brand"
        disabled={!!size(formErrors)}
        className="button-wrap"
        isLoading={isUpdating}
      >
        Save
      </Button>
    </form>
  );

  const renderContent = () => {
    switch (phase) {
      case 0:
        return renderForm();
      case 1:
        return (
          // TODO - confirm border color
          <InfoBanner>
            To complete your integration, follow the instructions in the other
            tab, then refresh this page to verify.
          </InfoBanner>
        );
      case 2:
        // checking integration
        return <Spinner />;
      case 3:
        return (
          // TODO - confirm border color
          <InfoBanner color="grey" className={`${baseClass}__success`}>
            <Icon name="success" />
            <b>Microsoft Entra tenant ID:</b>{" "}
            {/* TODO - address buginess with truncation –> tooltip enabling */}
            <TooltipTruncatedText value={formData[msetid]} />
          </InfoBanner>
        );
      default:
        return <Spinner />;
    }
  };
  return (
    <div className={baseClass}>
      <SectionHeader title="Conditional access" />
      <p className={`${baseClass}__page-description`}>
        Block hosts failing any policies from logging into third party apps.
        Enable or disable on the{" "}
        <CustomLink url={paths.MANAGE_POLICIES} text="Policies" /> page.
      </p>
      {renderContent()}
    </div>
  );
};

export default ConditionalAccess;
