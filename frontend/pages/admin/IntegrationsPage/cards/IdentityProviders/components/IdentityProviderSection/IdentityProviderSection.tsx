import React, { useContext } from "react";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import { dateAgo } from "utilities/date_format";
import { internationalTimeFormat } from "utilities/helpers";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";
import idpAPI from "services/entities/idp";

import SettingsSection from "pages/admin/components/SettingsSection";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import PageDescription from "components/PageDescription";
import EmptyState from "components/EmptyState";

import SectionCard from "../../../MdmSettings/components/SectionCard";

const baseClass = "identity-provider-section";

const AddEndUserInfoCard = () => {
  return (
    <EmptyState
      header="No IdP connected"
      info={
        <CustomLink
          text="Learn more"
          newTab
          url={`${LEARN_MORE_ABOUT_BASE_LINK}connect-idp`}
        />
      }
    />
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
          url={`${LEARN_MORE_ABOUT_BASE_LINK}troubleshoot-idp-connection`}
        />
      }
    >
      <p className={`${baseClass}__section-card-content`}>
        Received information from your IdP{" "}
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
          url={`${LEARN_MORE_ABOUT_BASE_LINK}troubleshoot-idp-connection`}
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
          Failed to receive information from your IdP ({dateAgo(receivedAt)}).
        </TooltipWrapper>
      </p>
    </SectionCard>
  );
};

const IdentityProviderSection = () => {
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
    <SettingsSection title="Identity provider (IdP)">
      {isPremiumTier && (
        <PageDescription
          content={
            <>
              Connect Fleet to your IdP to sync end user information (e.g.
              groups) to hosts.
            </>
          }
          variant="right-panel"
        />
      )}
      {renderContent()}
    </SettingsSection>
  );
};

export default IdentityProviderSection;
