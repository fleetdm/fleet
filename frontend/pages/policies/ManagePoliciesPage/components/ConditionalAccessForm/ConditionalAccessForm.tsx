import React, {
  forwardRef,
  useContext,
  useImperativeHandle,
  useState,
} from "react";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import PATHS from "router/paths";
import CustomLink from "components/CustomLink";
import Slider from "components/forms/fields/Slider";
import { AppContext } from "context/app";
import InfoBanner from "components/InfoBanner/InfoBanner";

const baseClass = "conditional-access-form";

export interface IConditionalAccessFormData {
  enabled: boolean;
}

export interface IConditionalAccessFormHandle {
  getFormData: () => IConditionalAccessFormData | null;
  validate: () => boolean;
  isDirty: () => boolean;
}

interface IConditionalAccessFormProps {
  configured: boolean;
  enabled: boolean;
  gitOpsModeEnabled?: boolean;
  providerText: string;
}

const ConditionalAccessForm = forwardRef<
  IConditionalAccessFormHandle,
  IConditionalAccessFormProps
>(
  (
    {
      configured,
      enabled,
      gitOpsModeEnabled = false,
      providerText,
    }: IConditionalAccessFormProps,
    ref
  ) => {
    const { isGlobalAdmin, isTeamAdmin } = useContext(AppContext);
    const isAdmin = isGlobalAdmin || isTeamAdmin;

    const [formEnabled, setFormEnabled] = useState(enabled);

    useImperativeHandle(ref, () => ({
      getFormData: () => (configured ? { enabled: formEnabled } : null),
      validate: () => true,
      isDirty: () => configured && formEnabled !== enabled,
    }));

    return (
      <div className={`${baseClass} form`}>
        <p className={`${baseClass}__description`}>
          Block single sign-on for end users failing policies.{" "}
          <CustomLink
            text="Learn more"
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/conditional-access`}
            newTab
          />
        </p>
        {!configured && (
          <InfoBanner className={baseClass}>
            To use conditional access automations, connect Fleet to{" "}
            {providerText} in{" "}
            <CustomLink
              url={PATHS.ADMIN_INTEGRATIONS_CONDITIONAL_ACCESS}
              text="Settings &gt; Integrations &gt; Conditional access"
              multiline
            />
            .
          </InfoBanner>
        )}
        {configured && (
          <Slider
            value={formEnabled}
            onChange={() => setFormEnabled(!formEnabled)}
            inactiveText="Disabled"
            activeText="Enabled"
            disabled={gitOpsModeEnabled || !isAdmin}
          />
        )}
      </div>
    );
  }
);

ConditionalAccessForm.displayName = "ConditionalAccessForm";

export default ConditionalAccessForm;
