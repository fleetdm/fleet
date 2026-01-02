import React, { useContext, useState, useEffect } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import classnames from "classnames";

import { ILabelSummary } from "interfaces/label";
import {
  IAppStoreApp,
  ISoftwarePackage,
  isSoftwarePackage,
  InstallerType,
} from "interfaces/software";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import softwareAPI, {
  MAX_FILE_SIZE_BYTES,
  MAX_FILE_SIZE_MB,
} from "services/entities/software";
import labelsAPI, { getCustomLabels } from "services/entities/labels";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import deepDifference from "utilities/deep_difference";
import { getFileDetails } from "utilities/file/fileUtils";

import Modal from "components/Modal";
import FileProgressModal from "components/FileProgressModal";
import CategoriesEndUserExperienceModal from "pages/SoftwarePage/components/modals/CategoriesEndUserExperienceModal";

import PackageForm from "pages/SoftwarePage/components/forms/PackageForm";
import { IPackageFormData } from "pages/SoftwarePage/components/forms/PackageForm/PackageForm";
import SoftwareVppForm from "pages/SoftwarePage/components/forms/SoftwareVppForm";
import { ISoftwareVppFormData } from "pages/SoftwarePage/components/forms/SoftwareVppForm/SoftwareVppForm";
import {
  generateSelectedLabels,
  getCustomTarget,
  getInstallType,
  getTargetType,
} from "pages/SoftwarePage/helpers";

import { getErrorMessage } from "./helpers";
import ConfirmSaveChangesModal from "../ConfirmSaveChangesModal";

const baseClass = "edit-software-modal";

// Install type used on add but not edit
export type IEditPackageFormData = Omit<IPackageFormData, "installType">;

interface IEditSoftwareModalProps {
  softwareId: number;
  teamId: number;
  softwareInstaller: ISoftwarePackage | IAppStoreApp;
  refetchSoftwareTitle: () => void;
  onExit: () => void;
  installerType: InstallerType;
  router: InjectedRouter;
  openViewYamlModal: () => void;
  isIosOrIpadosApp?: boolean;
  name: string;
  displayName: string;
  source?: string;
}

const EditSoftwareModal = ({
  softwareId,
  teamId,
  softwareInstaller,
  onExit,
  refetchSoftwareTitle,
  installerType,
  router,
  openViewYamlModal,
  isIosOrIpadosApp = false,
  name,
  displayName,
  source,
}: IEditSoftwareModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { config } = useContext(AppContext);

  const gitOpsModeEnabled = config?.gitops.gitops_mode_enabled || false;

  const [editSoftwareModalClasses, setEditSoftwareModalClasses] = useState(
    baseClass
  );
  const [isUpdatingSoftware, setIsUpdatingSoftware] = useState(false);
  const [
    showConfirmSaveChangesModal,
    setShowConfirmSaveChangesModal,
  ] = useState(false);
  const [
    showPreviewEndUserExperienceModal,
    setShowPreviewEndUserExperienceModal,
  ] = useState(false);

  const [
    pendingPackageUpdates,
    setPendingPackageUpdates,
  ] = useState<IEditPackageFormData>({
    software: null,
    installScript: "",
    selfService: false,
    automaticInstall: false,
    targetType: "",
    customTarget: "",
    labelTargets: {},
    categories: [],
  });
  const [
    pendingVppUpdates,
    setPendingVppUpdates,
  ] = useState<ISoftwareVppFormData>({
    selfService: false,
    automaticInstall: false,
    targetType: "",
    customTarget: "",
    labelTargets: {},
    categories: [],
  });
  const [uploadProgress, setUploadProgress] = useState(0);
  const [showFileProgressModal, setShowFileProgressModal] = useState(false);

  const { data: labels } = useQuery<ILabelSummary[], Error>(
    ["custom_labels"],
    () => labelsAPI.summary(teamId).then((res) => getCustomLabels(res.labels)),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
    }
  );

  // Work around to not lose Edit Software modal data when Save changes modal opens
  // by using CSS to hide Edit Software modal when Save changes modal is open
  useEffect(() => {
    setEditSoftwareModalClasses(
      classnames(baseClass, {
        [`${baseClass}--hidden`]:
          showConfirmSaveChangesModal ||
          showPreviewEndUserExperienceModal ||
          (!!pendingPackageUpdates.software && isUpdatingSoftware),
      })
    );
  }, [
    showConfirmSaveChangesModal,
    showPreviewEndUserExperienceModal,
    pendingPackageUpdates.software,
    isUpdatingSoftware,
  ]);

  /* 1. Delays showing the file progress modal until isUpdatingSoftware
   * has been true for 3 seconds to prevent flashing modal on quick uploads
   * 2. Prevents page unload during the upload
   * 3. Cleans both up when uploading stops or the component unmounts */
  useEffect(() => {
    // Timer for delayed modal
    let timeoutId: ReturnType<typeof setTimeout> | undefined;

    const beforeUnloadHandler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      // Next line with e.returnValue is included for legacy support
      // e.g.Chrome / Edge < 119
      e.returnValue = true;
    };

    if (isUpdatingSoftware) {
      // only show modal if still uploading after 3 seconds
      timeoutId = setTimeout(() => {
        setShowFileProgressModal(true);
      }, 3000);

      // Prevents user from leaving page while uploading
      addEventListener("beforeunload", beforeUnloadHandler);
    } else {
      // upload finished: hide modal and reset
      setShowFileProgressModal(false);
    }

    // Cleanup that runs when isUpdatingSoftware changes or component unmounts
    return () => {
      if (timeoutId) {
        clearTimeout(timeoutId);
      }
      removeEventListener("beforeunload", beforeUnloadHandler);
    };
  }, [isUpdatingSoftware]);

  // Close confirm modal when file progress modal opens
  useEffect(() => {
    if (showFileProgressModal) {
      setShowConfirmSaveChangesModal(false);
    }
  }, [showFileProgressModal]);

  const toggleConfirmSaveChangesModal = () => {
    setShowConfirmSaveChangesModal(!showConfirmSaveChangesModal);
  };

  const togglePreviewEndUserExperienceModal = () => {
    setShowPreviewEndUserExperienceModal(!showPreviewEndUserExperienceModal);
  };

  // Edit package API call
  const onEditPackage = async (formData: IEditPackageFormData) => {
    setIsUpdatingSoftware(true);

    if (formData.software && formData.software.size > MAX_FILE_SIZE_BYTES) {
      renderFlash(
        "error",
        `Couldn't edit software. The maximum file size is ${MAX_FILE_SIZE_MB} MB.`
      );
      setIsUpdatingSoftware(false);
      return;
    }

    try {
      await softwareAPI.editSoftwarePackage({
        data: formData,
        orignalPackage: softwareInstaller as ISoftwarePackage,
        softwareId,
        teamId,
        onUploadProgress: (progressEvent) => {
          const progress = progressEvent.progress || 0;
          // for large uploads it seems to take a bit for the server to finalize its response so we'll keep the
          // progress bar at 97% until the server response is received
          setUploadProgress(Math.max(progress - 0.03, 0.01));
        },
      });

      if (
        isSoftwarePackage(softwareInstaller) &&
        softwareInstaller.title_id &&
        gitOpsModeEnabled
      ) {
        // No longer flash message, we open YAML modal if editing with gitOpsModeEnabled
        openViewYamlModal();
      } else {
        renderFlash(
          "success",
          <>
            Successfully edited <b>{formData.software?.name}</b>.
            {formData.selfService
              ? " The end user can install from Fleet Desktop."
              : ""}
          </>
        );
      }
      refetchSoftwareTitle();
      onExit();
    } catch (e) {
      renderFlash(
        "error",
        getErrorMessage(e, softwareInstaller as IAppStoreApp)
      );
    }
    setIsUpdatingSoftware(false);
  };

  const isOnlySelfServiceUpdated = (updates: Record<string, any>) => {
    return Object.keys(updates).length === 1 && "selfService" in updates;
  };

  const onClickSavePackage = (formData: IPackageFormData) => {
    const softwarePackage = softwareInstaller as ISoftwarePackage;

    const currentData = {
      software: null,
      installScript: softwarePackage.install_script || "",
      preInstallQuery: softwarePackage.pre_install_query || "",
      postInstallScript: softwarePackage.post_install_script || "",
      uninstallScript: softwarePackage.uninstall_script || "",
      selfService: softwarePackage.self_service || false,
      installType: getInstallType(softwarePackage),
      targetType: getTargetType(softwarePackage),
      customTarget: getCustomTarget(softwarePackage),
      labelTargets: generateSelectedLabels(softwarePackage),
    };

    setPendingPackageUpdates(formData);

    const updates = deepDifference(formData, currentData);

    // Send an array with an empty string when all categories are unchecked
    // so that the "categories" key is included in the multipart form data and
    // will be deleted rather than ignored (an empty array would skip the field)
    if (!formData.categories?.length) {
      formData.categories = [""];
    }

    if (isOnlySelfServiceUpdated(updates)) {
      onEditPackage(formData);
    } else {
      setShowConfirmSaveChangesModal(true);
    }
  };

  // Edit App Store API call -- currently only for VPP apps and not Google Play apps
  const onEditVpp = async (formData: ISoftwareVppFormData) => {
    setIsUpdatingSoftware(true);

    try {
      await softwareAPI.editAppStoreApp(softwareId, teamId, formData);

      renderFlash(
        "success",
        <>
          Successfully edited <b>{softwareInstaller.name}</b>.
          {formData.selfService
            ? " The end user can install from Fleet Desktop."
            : ""}
        </>
      );
      onExit();
      refetchSoftwareTitle();
    } catch (e) {
      renderFlash(
        "error",
        getErrorMessage(e, softwareInstaller as IAppStoreApp)
      );
    }
    setIsUpdatingSoftware(false);
  };

  const onClickSaveVpp = async (formData: ISoftwareVppFormData) => {
    const currentData = {
      selfService: softwareInstaller.self_service || false,
      automaticInstall: softwareInstaller.automatic_install || false,
      targetType: getTargetType(softwareInstaller),
      customTarget: getCustomTarget(softwareInstaller),
      labelTargets: generateSelectedLabels(softwareInstaller),
    };

    setPendingVppUpdates(formData);

    const updates = deepDifference(formData, currentData);

    if (isOnlySelfServiceUpdated(updates)) {
      onEditVpp(formData);
    } else {
      setShowConfirmSaveChangesModal(true);
    }
  };

  const onClickConfirmChanges = () => {
    if (installerType === "package") {
      onEditPackage(pendingPackageUpdates);
    } else {
      onEditVpp(pendingVppUpdates);
    }
  };

  const renderForm = () => {
    if (installerType === "package") {
      const softwarePackage = softwareInstaller as ISoftwarePackage;
      return (
        <PackageForm
          labels={labels || []}
          className={`${baseClass}__package-form`}
          isEditingSoftware
          onCancel={onExit}
          onSubmit={onClickSavePackage}
          onClickPreviewEndUserExperience={togglePreviewEndUserExperienceModal}
          defaultSoftware={softwareInstaller}
          defaultInstallScript={softwarePackage.install_script}
          defaultPreInstallQuery={softwarePackage.pre_install_query}
          defaultPostInstallScript={softwarePackage.post_install_script}
          defaultUninstallScript={softwarePackage.uninstall_script}
          defaultSelfService={softwarePackage.self_service}
          defaultCategories={softwarePackage.categories}
        />
      );
    }

    return (
      <SoftwareVppForm
        labels={labels || []}
        softwareVppForEdit={softwareInstaller as IAppStoreApp}
        onSubmit={onClickSaveVpp}
        onCancel={onExit}
        isLoading={isUpdatingSoftware}
        onClickPreviewEndUserExperience={togglePreviewEndUserExperienceModal}
      />
    );
  };

  return (
    <>
      <Modal
        className={editSoftwareModalClasses}
        title={
          isSoftwarePackage(softwareInstaller) ? "Edit package" : "Edit app"
        }
        onExit={onExit}
        width="large"
      >
        {renderForm()}
      </Modal>
      {showConfirmSaveChangesModal && (
        <ConfirmSaveChangesModal
          onClose={toggleConfirmSaveChangesModal}
          softwareInstallerName={softwareInstaller?.name}
          installerType={installerType}
          onSaveChanges={onClickConfirmChanges}
          isLoading={isUpdatingSoftware}
        />
      )}
      {showPreviewEndUserExperienceModal && (
        <CategoriesEndUserExperienceModal
          name={name}
          displayName={displayName}
          source={source}
          iconUrl={softwareInstaller.icon_url || undefined}
          onCancel={togglePreviewEndUserExperienceModal}
          isIosOrIpadosApp={isIosOrIpadosApp}
        />
      )}
      {!!pendingPackageUpdates.software && showFileProgressModal && (
        <FileProgressModal
          fileDetails={getFileDetails(pendingPackageUpdates.software, true)}
          fileProgress={uploadProgress}
        />
      )}
    </>
  );
};

export default EditSoftwareModal;
