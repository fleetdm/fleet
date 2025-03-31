import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import { size } from "lodash";

import paths from "router/paths";

import { NotificationContext } from "context/notification";

import conditionalAccessAPI from "services/entities/conditional_access";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import CustomLink from "components/CustomLink";
import SectionHeader from "components/SectionHeader";

import { AppContext } from "context/app";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import { IFormField } from "interfaces/form_field";
import { IConfig } from "interfaces/config";
// import { getErrorReason } from "interfaces/errors";

const baseClass = "conditional-access";

const msetid = "microsoft_entra_tenant_id";

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

  const [formData, setFormData] = useState<IFormData>({
    [msetid]: "",
  });
  const [formErrors, setFormErrors] = useState<IFormErrors>({});
  const [isUpdating, setIsUpdating] = useState(false);

  // Redirect to /settings if not a cloud-managed Fleet instance. Must do this down at this level
  // since it depends on config context
  // if (!config.license?.managed_cloud) {
  //   // return <>OOPS</>;
  //   router.push(paths.ADMIN_SETTINGS);
  // }

  // TODO - actually call API
  setFormData({ [msetid]: "12345" });

  const handleSubmit = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const errs = validate(formData);
    if (Object.keys(errs).length > 0) {
      setFormErrors(errs);
      return;
    }
    setIsUpdating(true);
    try {
      await conditionalAccessAPI.triggerMicrosoftConditionalAccess(
        formData[msetid]
      );
      setIsUpdating(false);
      // TODO go to next step
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

  const renderContent = () => {
    return <>OOPS</>;
    if (!formData[msetid]) {
      return (
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
    }
    return <div>TODO</div>;
  };

  return <>OOPs2</>;
  return (
    <div className={baseClass}>
      <SectionHeader title="Conditional access" />
      <p className={`${baseClass}__page-description`}>
        Block hosts failing any policies from logging into third party apps.
        Enable or disable on the
        <CustomLink url={paths.MANAGE_POLICIES} text="Policies" />
        page.
      </p>
      {renderContent()}
    </div>
  );
};

export default ConditionalAccess;
