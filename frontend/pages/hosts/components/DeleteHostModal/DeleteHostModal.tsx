import React, { useContext } from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Modal from "components/Modal";
import { AppContext } from "context/app";
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
          <p>This may take up to 24 hours.</p>
        </>
      );
    }
    if (isIPadOrIPhone(platform)) {
      return (
        <>
          <p>This will remove all host data.</p>
          <p>This host will re-enroll unless MDM is turned off.</p>
        </>
      );
    }
    if (isMacOS(platform) && isMdmEnrolledInFleet) {
      if (useOneTimeEnrollSecrets) {
        return (
          <>
            <p>
              This will unenroll <b>{hostName}</b> but won&apos;t remove company
              data.
            </p>
            <p>
              To re-enroll it, wipe it or run{" "}
              <b>profiles renew -type enrollment</b> in the host&apos;s
              Terminal. {learnMoreLink}
            </p>
          </>
        );
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
            agent is uninstalled.{" "}
            <CustomLink
              text="Uninstall Fleet's agent"
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/uninstall-fleetd`}
              newTab
            />
          </li>
          <li>
            iOS, iPadOS, and Android hosts will re-appear unless MDM is turned
            off. For iOS and iPadOS it may take up to an hour and for Android it
            may take up to 24 hours to re-appear.
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
