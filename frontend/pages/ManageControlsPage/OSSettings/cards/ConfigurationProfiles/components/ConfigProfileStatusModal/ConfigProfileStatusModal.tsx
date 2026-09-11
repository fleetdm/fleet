import React, { useContext } from "react";
import { useQuery } from "react-query";

import configProfileAPI from "services/entities/config_profiles";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";

import Modal from "components/Modal";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import Button from "components/buttons/Button";
import { AppContext } from "context/app";
import CustomLink from "components/CustomLink";
import {
  isMDMConfiguredForPlatform,
  platformToMDMLabel,
  ProfilePlatform,
} from "interfaces/mdm";
import ConfigProfileStatusTable from "../ConfigProfileStatusTable";

const baseClass = "config-profile-status-modal";

interface IConfigProfileStatusModalProps {
  name: string;
  uuid: string;
  teamId: number;
  platform: ProfilePlatform;
  onClickResend: (hostCount: number) => void;
  onExit: () => void;
}

const ConfigProfileStatusModal = ({
  name,
  uuid,
  teamId,
  platform,
  onClickResend,
  onExit,
}: IConfigProfileStatusModalProps) => {
  const { config } = useContext(AppContext);
  const isMDMEnabled = isMDMConfiguredForPlatform(platform, config?.mdm);
  const { data, isLoading, isError } = useQuery(
    ["config-profile-status", uuid],
    () => configProfileAPI.getConfigProfileStatus(uuid),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isMDMEnabled,
    }
  );

  const renderContent = () => {
    if (!isMDMEnabled) {
      const mdmLabel = platformToMDMLabel(platform);
      let learnMoreUrl: string | undefined;
      // eslint-disable-next-line default-case
      switch (mdmLabel) {
        case "Apple":
          learnMoreUrl = `${LEARN_MORE_ABOUT_BASE_LINK}/turn-on-apple-mdm`;
          break;
        case "Windows":
          learnMoreUrl = `${LEARN_MORE_ABOUT_BASE_LINK}/setup-windows-mdm`;
          break;
        case "Android":
          learnMoreUrl = `${LEARN_MORE_ABOUT_BASE_LINK}/how-to-connect-android-enterprise`;
          break;
      }
      return (
        <DataError
          verticalPaddingSize="pad-medium"
          excludeIssueLink
          title={`${mdmLabel} MDM isn't turned on.`}
        >
          <span>
            Turn on {mdmLabel} MDM to manage this configuration profile.
            {learnMoreUrl ? (
              <>
                {" "}
                <CustomLink text="Learn more" newTab url={learnMoreUrl} />{" "}
              </>
            ) : null}
          </span>
        </DataError>
      );
    }

    if (isLoading) {
      return <Spinner />;
    }
    if (isError) {
      return <DataError verticalPaddingSize="pad-medium" />;
    }

    if (!data) {
      return null;
    }

    return (
      <ConfigProfileStatusTable
        teamId={teamId}
        uuid={uuid}
        profileStatus={data}
        onClickResend={onClickResend}
      />
    );
  };

  return (
    <Modal className={baseClass} title={name} onExit={onExit}>
      {renderContent()}
      <div className="modal-cta-wrap">
        <Button onClick={onExit}>Close</Button>
      </div>
    </Modal>
  );
};

export default ConfigProfileStatusModal;
