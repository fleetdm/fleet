import React, { useContext } from "react";
import { useQuery } from "react-query";

import idpAPI from "services/entities/idp";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { internationalTimeFormat } from "utilities/helpers";
import { dateAgo } from "utilities/date_format";
import { AppContext } from "context/app";

import SectionHeader from "components/SectionHeader";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";

import SectionCard from "../MdmSettings/components/SectionCard";

const baseClass = "identity-providers";

const AddEndUserInfoCard = () => {
  return (
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
        To add end user information, connect Fleet to Okta, Entra ID, or another
        identity provider (IdP).
      </p>
    </SectionCard>
  );
};

interface IReceivedEndUserInfoCardProps {
  receivedAt: string;
}

const ReceivedEndUserInfoCard = ({
  receivedAt,
}: IReceivedEndUserInfoCardProps) => {
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
          showArrow
          position="top"
          tipContent={internationalTimeFormat(new Date(receivedAt))}
          underline={false}
          className={`${baseClass}__received-tooltip`}
        >
          ({dateAgo(receivedAt)})
        </TooltipWrapper>
        .
      </p>
    </SectionCard>
  );
};

interface IFailedEndUserInfoCardProps {
  receivedAt: string;
  details: string;
}

const FailedEndUserInfoCard = ({
  receivedAt,
  details,
}: IFailedEndUserInfoCardProps) => {
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
          showArrow
          position="top"
          tipContent={`Error: ${details}`}
          underline={false}
          className={`${baseClass}__received-tooltip`}
        >
          Failed to receive end user information from your IdP (
          {dateAgo(receivedAt)}).
        </TooltipWrapper>
      </p>
    </SectionCard>
  );
};

const IdentityProviders = () => {
  const { isPremiumTier } = useContext(AppContext);

  const { data: scimIdPDetails, isLoading, isError } = useQuery(
    ["scim_details"],
    () => idpAPI.getSCIMDetails(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier,
    }
  );

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }

    if (isError) {
      return <DataError />;
    }

    if (isLoading) {
      return <Spinner />;
    }

    if (!scimIdPDetails) return null;

    if (scimIdPDetails.last_request === null) {
      return <AddEndUserInfoCard />;
    } else if (scimIdPDetails.last_request.status === "success") {
      return (
        <ReceivedEndUserInfoCard
          receivedAt={scimIdPDetails.last_request.requested_at}
        />
      );
    } else if (scimIdPDetails.last_request.status === "error") {
      return (
        <FailedEndUserInfoCard
          receivedAt={scimIdPDetails.last_request.requested_at}
          details={scimIdPDetails.last_request.details}
        />
      );
    }

    return null;
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Identity provider (IdP)" />
      {renderContent()}
    </div>
  );
};

export default IdentityProviders;
