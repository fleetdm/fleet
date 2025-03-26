import React from "react";

import SectionHeader from "components/SectionHeader";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";

import SectionCard from "../MdmSettings/components/SectionCard";

const baseClass = "identity-providers";

const AddEndUserInfoCard = () => {
  return (
    <div className={baseClass}>
      <SectionHeader title="Identity provider (IdP)" />
      <SectionCard
        header="Add end user information to your hosts"
        cta={
          <CustomLink
            text="Learn how"
            newTab
            url="https://fleetdm.com/learn-more-about/connect-idp"
            className={`${baseClass}__learn-more-link`}
          />
        }
      >
        <p className={`${baseClass}__section-card-content`}>
          To add end user information, connect Fleet to Okta, Entra ID, or
          another identity provider (IdP).
        </p>
      </SectionCard>
    </div>
  );
};

const RecievedEndUserInfoCard = () => {
  return (
    <SectionCard
      iconName="success"
      cta={
        <CustomLink
          text="Learn more"
          newTab
          url="https://fleetdm.com/learn-more-about/troubleshoot-idp-connection"
          className={`${baseClass}__learn-more-link`}
        />
      }
    >
      <p className={`${baseClass}__section-card-content`}>
        Received end user information from your IdP{" "}
        <TooltipWrapper
          tipContent="some date"
          underline={false}
          className={`${baseClass}__recieved-tooltip`}
        >
          (2 days ago)
        </TooltipWrapper>
        .
      </p>
    </SectionCard>
  );
};

const FailedEndUserInfoCard = () => {
  return (
    <SectionCard
      iconName="error"
      cta={
        <CustomLink
          text="Learn more"
          newTab
          url="https://fleetdm.com/learn-more-about/troubleshoot-idp-connection"
          className={`${baseClass}__learn-more-link`}
        />
      }
    >
      <p className={`${baseClass}__section-card-content`}>
        <TooltipWrapper
          tipContent='Error: Missing required attributes. "userName", "givenName", and "familyName" are required. Please configure your identity provider to send required attributes to Fleet.'
          underline={false}
          className={`${baseClass}__recieved-tooltip`}
        >
          Failed to receive end user information from your IdP (2 days ago).
        </TooltipWrapper>
      </p>
    </SectionCard>
  );
};

interface IIdentityProvidersProps {}

const IdentityProviders = ({}: IIdentityProvidersProps) => {
  return <FailedEndUserInfoCard />;
};

export default IdentityProviders;
