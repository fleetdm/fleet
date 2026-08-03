import React, { useCallback, useState } from "react";

import deviceUserAPI from "services/entities/device_user";

import { notify } from "components/ToastNotification";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import InstallAllInCategoryModal from "./InstallAllInCategoryModal";

const baseClass = "install-all-in-category-button";

export interface IInstallAllInCategoryButtonProps {
  /** Number of items in the selected category that are not yet installed.
   * Does not include software that has INSTALLED_OR_IN_FLIGHT_UI_STATUSES */
  uninstalledCount: number;
  /** True if any item in the selected category is currently in-progress. */
  hasInProgressInCategory: boolean;
  deviceToken: string;
  /** ID of the currently selected category. Undefined when "All" is selected
   * — the service omits the `category_id` query param and the BE installs
   * every uninstalled item the device user is entitled to. */
  categoryId?: number;
  /** Called after the install_all request resolves successfully. */
  onSuccess: () => void;
}

const InstallAllInCategoryButton = ({
  uninstalledCount,
  hasInProgressInCategory,
  deviceToken,
  categoryId,
  onSuccess,
}: IInstallAllInCategoryButtonProps) => {
  const [showModal, setShowModal] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleConfirm = useCallback(async () => {
    setIsSubmitting(true);
    try {
      await deviceUserAPI.installAllSelfServiceSoftwareInCategory(
        deviceToken,
        categoryId
      );
      setShowModal(false);
      onSuccess();
    } catch (error) {
      notify.error("Couldn't install. Please try again.", { response: error });
    } finally {
      setIsSubmitting(false);
    }
  }, [deviceToken, categoryId, onSuccess]);

  // Nothing eligible and no install_all batch running — drop the button from
  // the DOM. When a previous batch IS still running (count === 0 &&
  // hasInProgressInCategory), fall through and render a disabled "Install all"
  // (no count) so the user keeps a visual anchor on the action they triggered
  // until items settle.
  if (uninstalledCount === 0 && !hasInProgressInCategory) {
    return null;
  }

  // `count === 0` only reaches this line during an in-flight batch (the
  // early-return handles count=0 with no batch). That's the one and only
  // state where the button renders disabled.
  const isDisabled = uninstalledCount === 0;
  const label =
    uninstalledCount === 0
      ? "Install all"
      : `Install all (${uninstalledCount})`;

  return (
    <>
      <Button
        className={baseClass}
        variant="secondary"
        onClick={() => setShowModal(true)}
        disabled={isDisabled}
      >
        <Icon name="install" color="ui-fleet-black-75" />
        {label}
      </Button>
      {showModal && (
        <InstallAllInCategoryModal
          count={uninstalledCount}
          isSubmitting={isSubmitting}
          onConfirm={handleConfirm}
          onExit={() => setShowModal(false)}
        />
      )}
    </>
  );
};

export default InstallAllInCategoryButton;
