import React, { useContext, useState } from "react";
import classnames from "classnames";

import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import hostAPI from "services/entities/hosts";
import { NotificationContext } from "context/notification";

import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import { IHostMdmProfileWithAddedStatus } from "../OSSettingsTableConfig";
import { isDiskEncryptionProfile } from "../OSSettingStatusCell/helpers";

const baseClass = "os-settings-error-cell";

interface IRefetchButtonProps {
  isFetching: boolean;
  onClick: (evt: React.MouseEvent<HTMLButtonElement, React.MouseEvent>) => void;
}

const RefetchButton = ({ isFetching, onClick }: IRefetchButtonProps) => {
  const classNames = classnames(`${baseClass}__resend-button`, {
    [`${baseClass}__resending`]: isFetching,
  });

  const buttonText = isFetching ? "Resending..." : "Resend";

  // add additonal props when we need to display a tooltip for the button

  return (
    <Button
      disabled={isFetching}
      onClick={onClick}
      variant="text-icon"
      className={classNames}
    >
      <Icon name="refresh" color="core-fleet-blue" size="small" />
      {buttonText}
    </Button>
  );
};

/**
 * generates the formatted tooltip for the error column.
 * the expected format of the error string is:
 * "key1: value1, key2: value2, key3: value3"
 */
const generateFormattedTooltip = (detail: string) => {
  const keyValuePairs = detail.split(/, */);
  const formattedElements: JSX.Element[] = [];

  // Special case to handle bitlocker error message. It does not follow the
  // expected string format so we will just render the error message as is.
  if (
    detail.includes("BitLocker") ||
    detail.includes("preparing volume for encryption")
  ) {
    return detail;
  }

  keyValuePairs.forEach((pair, i) => {
    const [key, value] = pair.split(/: */);
    if (key && value) {
      formattedElements.push(
        <span key={key}>
          <b>{key.trim()}:</b> {value.trim()}
          {/* dont add the trailing comma for the last element */}
          {i !== keyValuePairs.length - 1 && (
            <>
              ,<br />
            </>
          )}
        </span>
      );
    }
  });

  return formattedElements.length ? <>{formattedElements}</> : detail;
};

/**
 * generates the error tooltip for the error column. This will be formatted or
 * unformatted.
 */
const generateErrorTooltip = (
  cellValue: string,
  profile: IHostMdmProfileWithAddedStatus
) => {
  if (profile.status !== "failed") return null;

  if (profile.platform !== "windows") {
    return cellValue;
  }
  return generateFormattedTooltip(profile.detail);
};

interface IOSSettingsErrorCellProps {
  hostId?: number;
  profile: IHostMdmProfileWithAddedStatus;
  onProfileResent?: () => void;
}

const OSSettingsErrorCell = ({
  hostId,
  profile,
  onProfileResent,
}: IOSSettingsErrorCellProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isLoading, setIsLoading] = useState(false);

  const onResendProfile = async () => {
    if (!hostId) return;
    setIsLoading(true);
    try {
      await hostAPI.resendProfile(hostId, profile.profile_uuid);
      onProfileResent?.();
    } catch (e) {
      renderFlash("error", "Couldn't resend. Please try again.");
    }
    onProfileResent?.();
    setIsLoading(false);
  };

  // const isFailed = profile.status === "failed";
  const isFailed = true;
  const isVerified = profile.status === "verified";
  const showRefetchButton =
    (isFailed || isVerified) &&
    !isDiskEncryptionProfile(profile.name) &&
    hostId !== undefined;
  const value = (isFailed && profile.detail) || DEFAULT_EMPTY_CELL_VALUE;

  const tooltip = generateErrorTooltip(value, profile);

  return (
    <div className={baseClass}>
      <TooltipTruncatedTextCell
        tooltipBreakOnWord
        tooltip={tooltip}
        value={value}
        classes={showRefetchButton ? `${baseClass}__failed-message` : undefined}
      />
      {showRefetchButton && (
        <RefetchButton isFetching={isLoading} onClick={onResendProfile} />
      )}
    </div>
  );
};

export default OSSettingsErrorCell;
