import React, { useCallback, useContext, useState } from "react";

import deviceUserAPI from "services/entities/device_user";
import { NotificationContext } from "context/notification";

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
  const { renderFlash } = useContext(NotificationContext);
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
      renderFlash("error", "Couldn't install. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  }, [deviceToken, categoryId, onSuccess, renderFlash]);

  const isDisabled = hasInProgressInCategory || uninstalledCount === 0;

  return (
    <>
      <Button
        className={baseClass}
        variant="inverse"
        onClick={() => setShowModal(true)}
        disabled={isDisabled}
      >
        <Icon name="install" color="ui-fleet-black-75" />
        Install all ({uninstalledCount})
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
