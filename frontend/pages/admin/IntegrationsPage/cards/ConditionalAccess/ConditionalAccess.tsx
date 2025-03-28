import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import { NotificationContext } from "context/notification";

import paths from "router/paths";

import CustomLink from "components/CustomLink";
import SectionHeader from "components/SectionHeader";
import { AppContext } from "context/app";

const baseClass = "conditional-access";

interface IFormData {}

interface IFormErrors {}

const validate = (formData: IFormData) => {
  const errs: IFormErrors = {};
  return errs;
};

const ConditionalAccess = (router: InjectedRouter) => {
  const { renderFlash } = useContext(NotificationContext);
  const { config } = useContext(AppContext);

  const [formData, setFormData] = useState<IConditionalAccessFormData>({});
  const [formErrors, setFormErrors] = useState<IConditionalAccessFormErrors>(
    {}
  );
  const [isUpdating, setIsUpdating] = useState(false);

  // Redirect to /settings if not a cloud-managed Fleet instance. Must do this down at this level
  // since it depends on config context
  if (!config) return;
  if (!config.license.managed_cloud) {
    router.push(paths.ADMIN_SETTINGS);
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
    } catch (e) {}
  };

  const onInputChange = ({ name, value }: IFormField) => {
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
    // TODO
    return null;
    // <form onSubmit={handleSubmit}>
    //   <Checkbox
    //     onChange={onInputChange}
    //     name="gitOpsModeEnabled"
    //     value={gitOpsModeEnabled}
    //     parseTarget
    //   >
    //     <TooltipWrapper tipContent="GitOps mode is a UI-only setting. API permissions are restricted based on user role.">
    //       Enable GitOps mode
    //     </TooltipWrapper>
    //   </Checkbox>
    //   {/* Git repository URL */}
    //   <InputField
    //     label="Git repository URL"
    //     onChange={onInputChange}
    //     name="repoURL"
    //     value={repoURL}
    //     parseTarget
    //     onBlur={onInputBlur}
    //     error={formErrors.repository_url}
    //     helpText="When GitOps mode is enabled, you will be directed here to make changes."
    //     disabled={!gitOpsModeEnabled}
    //   />
    //   <Button
    //     type="submit"
    //     disabled={!!Object.keys(formErrors).length}
    //     isLoading={isUpdating}
    //   >
    //     Save
    //   </Button>
    // </form>
  };

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
