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
  isLinuxLike,
  isMacOS,
  isWindows,
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
  /** Display name of the host when exactly one host is being deleted. */
  hostName?: string;
  /** Set when every host being deleted shares a platform, so the modal can
   * show the per-platform copy instead of the generic bulk copy. */
  platform?: HostPlatform;
  isMdmEnrolledInFleet?: boolean;
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

  const getCount = () => {
    if (!selectedHostIds) {
      return 1;
    }
    if (isAllMatchingHostsSelected && hostsCount !== undefined) {
      return hostsCount;
    }
    return selectedHostIds.length;
  };
  const count = getCount();
  const isPlural = count !== 1;

  const hostText = () => {
    if (hostName) {
      return hostName;
    }
    const suffix =
      isAllMatchingHostsSelected && hostsCount === undefined ? "+" : "";
    return `${count}${suffix} ${strUtils.pluralize(count, "host")}`;
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

  const theseHosts = isPlural ? "These hosts" : "This host";
  const them = isPlural ? "them" : "it";

  // Plural selections lead with the count so admins can see what they are
  // deleting; the singular reads as the host page copy from the design.
  const removeAllDataSentence = (details: string) =>
    isPlural ? (
      <>
        This will remove <b>{hostText()}</b> and associated data{details}.
      </>
    ) : (
      <>This will remove all host data{details}.</>
    );

  const renderMacOneTimeSecretBody = () => {
    // Hosts enrolled through Apple Business come back as pending hosts and
    // can renew their own enrollment. Manually enrolled hosts need fleetd
    // reinstalled to get a new one-time secret.
    const reEnrollInstructions =
      mdmEnrollmentStatus === "On (automatic)" ? (
        <>
          To re-enroll {them}, wipe {them} or run{" "}
          <b>sudo profiles renew -type enrollment</b> in{" "}
          {isPlural ? "each host's" : "the host's"} Terminal.
        </>
      ) : (
        <>
          To re-enroll {them}, turn on MDM manually or reinstall Fleet&apos;s
          agent.
        </>
      );
    return (
      <>
        <p>
          This will unenroll <b>{hostText()}</b> but won&apos;t remove company
          data. {learnMoreLink}
        </p>
        <p>{reEnrollInstructions}</p>
      </>
    );
  };

  const renderPlatformBody = () => {
    if (!platform) {
      return null;
    }
    if (isAndroid(platform)) {
      return (
        <>
          <p>
            This will unenroll <b>{hostText()}</b> and remove company data.
          </p>
          <p>This may take up to 24 hours. {learnMoreLink}</p>
        </>
      );
    }
    if (isIPadOrIPhone(platform)) {
      return (
        <>
          <p>{removeAllDataSentence("")}</p>
          <p>
            {theseHosts} will re-enroll unless MDM is turned off.{" "}
            {learnMoreLink}
          </p>
        </>
      );
    }
    if (isMacOS(platform) && isMdmEnrolledInFleet && useOneTimeEnrollSecrets) {
      return renderMacOneTimeSecretBody();
    }
    if (isMacOS(platform) || isWindows(platform) || isLinuxLike(platform)) {
      return (
        <>
          <p>
            {removeAllDataSentence(
              " such as unlock PINs and disk encryption keys"
            )}
          </p>
          <p>
            {theseHosts} will re-enroll unless Fleet&apos;s agent is
            uninstalled. {learnMoreLink}
          </p>
        </>
      );
    }
    return null;
  };

  const renderBody = () =>
    renderPlatformBody() ?? (
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
