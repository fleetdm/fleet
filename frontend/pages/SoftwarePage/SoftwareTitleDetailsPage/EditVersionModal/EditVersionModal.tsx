import React, { useState } from "react";
import { useQuery, useQueryClient } from "react-query";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import { getErrorReason } from "interfaces/errors";
import { ILabelSummary } from "interfaces/label";
import { IAppStoreAppVersion } from "interfaces/software";
import CategoriesEndUserExperienceModal from "pages/SoftwarePage/components/modals/CategoriesEndUserExperienceModal";
import labelsAPI, { getCustomLabels } from "services/entities/labels";
import softwareAPI from "services/entities/software";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import VersionFormFields, {
  IVersionFormData,
  versionToFormData,
} from "../VersionFormFields";

const baseClass = "edit-version-modal";

interface IEditVersionModalProps {
  softwareId: number;
  teamId: number;
  version: IAppStoreAppVersion;
  /** Other version names on this title, used for client-side uniqueness
   * validation. Excludes the version being edited. */
  siblingVersionNames: string[];
  onExit: () => void;
  onSuccess: () => void;
}

const buildLabelArray = (labelTargets: Record<string, boolean>): string[] =>
  Object.entries(labelTargets)
    .filter(([, selected]) => selected)
    .map(([name]) => name);

const EditVersionModal = ({
  softwareId,
  teamId,
  version,
  siblingVersionNames,
  onExit,
  onSuccess,
}: IEditVersionModalProps) => {
  const queryClient = useQueryClient();

  // Empty-config scaffold shown in the editor. A submit equal to this value
  // is treated as "no configuration".
  const isIosOrIpados =
    version.platform === "ios" || version.platform === "ipados";
  const EMPTY_XML_SCAFFOLD = "<dict>\n  \n</dict>";
  const EMPTY_JSON_SCAFFOLD = "{}";
  const emptyScaffold = isIosOrIpados
    ? EMPTY_XML_SCAFFOLD
    : EMPTY_JSON_SCAFFOLD;

  const siblingNamesSet = new Set(
    siblingVersionNames.map((n) => n.toLowerCase())
  );

  const [serverErrors, setServerErrors] = useState<IFormErrors | null>(null);
  const [
    showPreviewEndUserExperience,
    setShowPreviewEndUserExperience,
  ] = useState(false);

  const validate = (data: IVersionFormData): IFormErrors => {
    const errors: IFormErrors = {};
    const trimmed = data.name.trim();
    if (!trimmed) {
      errors.name = "Enter a version name";
    } else if (siblingNamesSet.has(trimmed.toLowerCase())) {
      errors.name = "A version with this name already exists on this fleet";
    }
    return errors;
  };

  const initialFormData = (() => {
    const base = versionToFormData(version);
    // Pre-fill the editor with the scaffold when the stored config is empty.
    if (!base.configuration) base.configuration = emptyScaffold;
    return base;
  })();

  const {
    formData,
    setField,
    commitFields,
    getError,
    clearFieldError,
    validateField,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IVersionFormData>({
    initialFormData,
    validate,
    serverErrors,
  });

  const { data: labels } = useQuery<ILabelSummary[], Error>(
    ["custom_labels", teamId],
    () => labelsAPI.summary(teamId).then((res) => getCustomLabels(res.labels)),
    { ...DEFAULT_USE_QUERY_OPTIONS }
  );

  const onValidSubmit = async (data: IVersionFormData) => {
    // Target → always send the three label arrays so the backend can
    // normalize. "All hosts" sends empty arrays to clear any existing
    // label scope. "Custom" sends the active list on the active key and
    // empty arrays on the other two.
    const activeLabels =
      data.targetType === "Custom" ? buildLabelArray(data.labelTargets) : [];
    const labelsIncludeAny =
      data.targetType === "Custom" && data.customTarget === "labelsIncludeAny"
        ? activeLabels
        : [];
    const labelsIncludeAll =
      data.targetType === "Custom" && data.customTarget === "labelsIncludeAll"
        ? activeLabels
        : [];
    const labelsExcludeAny =
      data.targetType === "Custom" && data.customTarget === "labelsExcludeAny"
        ? activeLabels
        : [];

    try {
      await softwareAPI.editAppStoreAppVersion(softwareId, teamId, version.id, {
        name: data.name,
        self_service: data.selfService,
        categories: data.categories,
        configuration:
          data.configuration && data.configuration !== emptyScaffold
            ? data.configuration
            : "",
        labels_include_any: labelsIncludeAny,
        labels_include_all: labelsIncludeAll,
        labels_exclude_any: labelsExcludeAny,
        auto_update_enabled: data.autoUpdateEnabled,
        auto_update_window_start:
          data.autoUpdateEnabled && data.autoUpdateWindowStart
            ? data.autoUpdateWindowStart
            : undefined,
        auto_update_window_end:
          data.autoUpdateEnabled && data.autoUpdateWindowEnd
            ? data.autoUpdateWindowEnd
            : undefined,
      });

      notify.success(
        <>
          Successfully edited <strong>{data.name}</strong>.
          {data.selfService
            ? " The end user can install from Fleet Desktop."
            : ""}
        </>
      );
      queryClient.invalidateQueries({
        queryKey: [{ scope: "software-titles" }],
      });
      queryClient.invalidateQueries({
        queryKey: [{ scope: "software-library" }],
      });
      onSuccess();
    } catch (e) {
      const reason = getErrorReason(e);
      if (reason?.toLowerCase().includes("name")) {
        setServerErrors({ name: reason });
      } else {
        notify.error("Couldn't edit. Please try again.", { response: e });
      }
    }
  };

  return (
    <>
      <Modal
        className={baseClass}
        title="Edit version"
        onExit={onExit}
        width="large"
      >
        <form
          className={`${baseClass}__form`}
          onSubmit={handleSubmit(onValidSubmit)}
        >
          <VersionFormFields
            formData={formData}
            setField={setField}
            commitFields={commitFields}
            getError={getError}
            clearFieldError={clearFieldError}
            validateField={validateField}
            platform={version.platform}
            appDisplayName={version.display_name || version.name}
            labels={labels ?? []}
            onClickPreviewEndUserExperience={
              version.platform !== "android"
                ? () => setShowPreviewEndUserExperience(true)
                : undefined
            }
          />

          <div className="modal-cta-wrap">
            <Button
              type="submit"
              isLoading={isSubmitting}
              disabled={isSubmitting}
            >
              Save
            </Button>
            <Button
              onClick={onExit}
              variant="secondary"
              disabled={isSubmitting}
            >
              Cancel
            </Button>
          </div>
        </form>
      </Modal>
      {showPreviewEndUserExperience && (
        <CategoriesEndUserExperienceModal
          onCancel={() => setShowPreviewEndUserExperience(false)}
          teamId={teamId}
          isIosOrIpadosApp={version.platform !== "android"}
        />
      )}
    </>
  );
};

export default EditVersionModal;
