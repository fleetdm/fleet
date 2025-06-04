import React from "react";
import { useQuery } from "react-query";

import idpAPI from "services/entities/idp";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { internationalTimeFormat } from "utilities/helpers";
import { dateAgo } from "utilities/date_format";

import SectionHeader from "components/SectionHeader";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

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

interface IRecievedEndUserInfoCardProps {
  recievedAt: string;
}

const RecievedEndUserInfoCard = ({
  recievedAt,
}: IRecievedEndUserInfoCardProps) => {
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
          tipContent={internationalTimeFormat(new Date(recievedAt))}
          underline={false}
          className={`${baseClass}__recieved-tooltip`}
        >
          ({dateAgo(recievedAt)})
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
          className={`${baseClass}__recieved-tooltip`}
        >
          Failed to receive end user information from your IdP (
          {dateAgo(receivedAt)}).
        </TooltipWrapper>
      </p>
    </SectionCard>
  );
};

const IdentityProviders = () => {
  const { data, isLoading, isError } = useQuery(
    ["scim_details"],
    () => idpAPI.getSCIMDetails(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
    }
  );

  const renderContent = () => {
    if (isError) {
      return <DataError />;
    }

    if (isLoading) {
      return <Spinner />;
    }

    if (!data) return null;

    if (data.last_request === null) {
      return <AddEndUserInfoCard />;
    } else if (data.last_request.status === "success") {
      return (
        <RecievedEndUserInfoCard recievedAt={data.last_request.requested_at} />
      );
    } else if (data.last_request.status === "error") {
      return (
        <FailedEndUserInfoCard
          receivedAt={data.last_request.requested_at}
          details={data.last_request.details}
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
