import React, { useContext } from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Modal from "components/Modal";
import { AppContext } from "context/app";
import { MdmEnrollmentStatus } from "interfaces/mdm";
import {
  HostPlatform,
  isAndroid,
  isIPadOrIPhone,
  isMacOS,
} from "interfaces/platform";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import strUtils from "utilities/strings";

const baseClass = "delete-host-modal";

interface IDeleteHostModalProps {
  onSubmit: () => void;
  onCancel: () => void;
  /** Manage host page only */
  isAllMatchingHostsSelected?: boolean;
  /** Manage host page only */
  selectedHostIds?: number[];
  /** Manage host page only */
  hostsCount?: number;
  /** Host details page only */
  hostName?: string;
  /** Host details page only */
  platform?: HostPlatform;
  /** Host details page only */
  isMdmEnrolledInFleet?: boolean;
  /** Host details page only */
  mdmEnrollmentStatus?: MdmEnrollmentStatus | null;
  isUpdating: boolean;
}

const DeleteHostModal = ({
  onSubmit,
  onCancel,
  isAllMatchingHostsSelected,
  selectedHostIds,
  hostsCount,
  hostName,
  platform,
  isMdmEnrolledInFleet,
  mdmEnrollmentStatus,
  isUpdating,
}: IDeleteHostModalProps): JSX.Element => {
  const { config } = useContext(AppContext);
  const useOneTimeEnrollSecrets = !!config?.auth?.use_one_time_enroll_secrets;

  const hostText = () => {
    if (selectedHostIds) {
      const count =
        isAllMatchingHostsSelected && hostsCount !== undefined
          ? hostsCount
          : selectedHostIds.length;
      const suffix =
        isAllMatchingHostsSelected && hostsCount === undefined ? "+" : "";
      return `${count}${suffix} ${strUtils.pluralize(count, "host")}`;
    }
    return hostName;
  };

  const hasManyHosts =
    selectedHostIds &&
    isAllMatchingHostsSelected &&
    hostsCount &&
    hostsCount >= 500;

  const learnMoreLink = (
    <CustomLink
      text="Learn more"
      url={`${LEARN_MORE_ABOUT_BASE_LINK}/deleting-a-host`}
      newTab
    />
  );

  const renderMacOneTimeSecretBody = () => {
    // Hosts enrolled through Apple Business come back as pending hosts and
    // can renew their own enrollment. Manually enrolled hosts need fleetd
    // reinstalled to get a new one-time secret.
    const reEnrollInstructions =
      mdmEnrollmentStatus === "On (automatic)" ? (
        <>
          To re-enroll it, wipe it or run <b>profiles renew -type enrollment</b>{" "}
          in the host&apos;s Terminal. {learnMoreLink}
        </>
      ) : (
        <>To re-enroll it, Fleet&apos;s agent must be reinstalled.</>
      );
    return (
      <>
        <p>
          This will unenroll <b>{hostName}</b> but won&apos;t remove company
          data.
        </p>
        <p>{reEnrollInstructions}</p>
      </>
    );
  };

  const renderSingleMdmHostBody = () => {
    if (selectedHostIds || !platform) {
      return null;
    }
    if (isAndroid(platform)) {
      return (
        <>
          <p>
            This will unenroll <b>{hostName}</b> and remove company data.
          </p>
          <p>This may take up to 24 hours. {learnMoreLink}</p>
        </>
      );
    }
    if (isIPadOrIPhone(platform)) {
      return (
        <>
          <p>This will remove all host data.</p>
          <p>
            This host will re-enroll unless MDM is turned off. {learnMoreLink}
          </p>
        </>
      );
    }
    if (isMacOS(platform) && isMdmEnrolledInFleet) {
      if (useOneTimeEnrollSecrets) {
        return renderMacOneTimeSecretBody();
      }
      return (
        <>
          <p>
            This will remove all host data such as unlock PINs and disk
            encryption keys.
          </p>
          <p>
            This host will re-enroll unless Fleet&apos;s agent is uninstalled.{" "}
            {learnMoreLink}
          </p>
        </>
      );
    }
    return null;
  };

  const renderBody = () =>
    renderSingleMdmHostBody() ?? (
      <>
        <p>
          This will remove <b>{hostText()}</b> and associated data such as
          unlock PINs and disk encryption keys.
        </p>
        {hasManyHosts && (
          <p>
            When deleting a large volume of hosts, it may take some time for
            this change to be reflected in the UI.
          </p>
        )}
        <ul>
          <li>
            macOS, Windows, or Linux hosts will re-appear unless Fleet&apos;s
            agent is uninstalled. {learnMoreLink}
          </li>
          <li>
            iOS and iPadOS will re-enroll unless MDM is turned off. Android will
            remove company data and may take up to 24 hours.
          </li>
        </ul>
      </>
    );

  return (
    <Modal title="Delete" onExit={onCancel} className={baseClass}>
      {renderBody()}
      <div className="modal-cta-wrap">
        <Button
          type="button"
          onClick={onSubmit}
          variant="alert"
          className="delete-loading"
          isLoading={isUpdating}
        >
          Delete
        </Button>
        <Button onClick={onCancel} variant="secondary">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default DeleteHostModal;
