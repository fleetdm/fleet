import React from "react";
import { useQuery } from "react-query";

import Button from "components/buttons/Button";
import FormField from "components/forms/FormField";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import {
  ILabelConfig,
  LabelTargetMode,
  TargetLabelSelector,
  TargetType,
} from "components/TargetLabelSelector";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import { ILabelSummary } from "interfaces/label";
import labelsAPI, {
  getCustomLabels,
  listNamesFromSelectedLabels,
} from "services/entities/labels";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

const baseClass = "edit-targets-modal";

export interface IProfileTargets {
  targetType: TargetType;
  includeMode: LabelTargetMode;
  includeLabels: Record<string, boolean>;
  excludeLabels: Record<string, boolean>;
}

const validate = (targets: IProfileTargets): IFormErrors => {
  const errors: IFormErrors = {};
  if (
    targets.targetType === "Custom" &&
    listNamesFromSelectedLabels(targets.includeLabels).length < 1 &&
    listNamesFromSelectedLabels(targets.excludeLabels).length < 1
  ) {
    errors.target = "Select at least one label";
  }
  return errors;
};

interface IEditTargetsModalProps {
  fleetId: number;
  targets: IProfileTargets;
  /** Shows the targets without letting them change. GitOps mode is always
   * read-only. */
  disabled?: boolean;
  onAddLabel: () => void;
  onDone: (targets: IProfileTargets) => void;
  onExit: () => void;
}

/** Edits a copy of the targets, so Cancel leaves the profile's untouched. */
const EditTargetsModal = ({
  fleetId,
  targets,
  disabled = false,
  onAddLabel,
  onDone,
  onExit,
}: IEditTargetsModalProps) => {
  const {
    formData: draft,
    commitFields,
    getError,
    handleSubmit,
  } = useFormValidation<IProfileTargets>({
    initialFormData: targets,
    validate,
  });

  const {
    data: labels,
    isLoading: isLoadingLabels,
    isError: isErrorLabels,
  } = useQuery<ILabelSummary[], Error>(
    ["custom_labels", fleetId],
    () => labelsAPI.summary(fleetId).then((res) => getCustomLabels(res.labels)),
    DEFAULT_USE_QUERY_OPTIONS
  );

  const includeTab: ILabelConfig = {
    selectedLabels: draft.includeLabels,
    onSelectLabel: ({ name, value }) =>
      commitFields({
        includeLabels: { ...draft.includeLabels, [name]: value },
      }),
    showModeToggle: true,
    mode: draft.includeMode,
    onSelectMode: (includeMode) => commitFields({ includeMode }),
    anyTooltip: (
      <>
        Profile will be applied to hosts that{" "}
        <em>
          <b>have any</b>
        </em>{" "}
        of these labels.
      </>
    ),
    allTooltip: (
      <>
        Profile will be applied to hosts that{" "}
        <em>
          <b>have all</b>
        </em>{" "}
        of these labels.
      </>
    ),
  };

  const excludeTab: ILabelConfig = {
    selectedLabels: draft.excludeLabels,
    onSelectLabel: ({ name, value }) =>
      commitFields({
        excludeLabels: { ...draft.excludeLabels, [name]: value },
      }),
  };

  // Enter in the label search would otherwise submit the modal; it only
  // filters.
  const onKeyDown = (evt: React.KeyboardEvent<HTMLDivElement>) => {
    if (
      evt.key === "Enter" &&
      evt.target instanceof HTMLInputElement &&
      evt.target.type === "text"
    ) {
      evt.preventDefault();
    }
  };

  return (
    <Modal className={baseClass} title="Edit targets" onExit={onExit}>
      <form onSubmit={handleSubmit(onDone)}>
        <div onKeyDown={onKeyDown}>
          <GitOpsModeTooltipWrapper
            isInputField
            renderChildren={(disableChildren) => (
              <FormField
                label="Target"
                name="target"
                error={getError("target")}
              >
                <TargetLabelSelector
                  selectedTargetType={draft.targetType}
                  onSelectTargetType={(targetType) =>
                    commitFields({ targetType })
                  }
                  labels={labels || []}
                  includeConfig={includeTab}
                  excludeConfig={excludeTab}
                  isLoadingLabels={isLoadingLabels}
                  isErrorLabels={isErrorLabels}
                  emptyStateDescription="Add a label to target your configuration profile."
                  onAddLabel={onAddLabel}
                  disableOptions={disabled || disableChildren}
                />
              </FormField>
            )}
          />
        </div>
        <ModalFooter
          primaryButtons={
            <>
              <Button variant="secondary" onClick={onExit}>
                Cancel
              </Button>
              <GitOpsModeTooltipWrapper
                renderChildren={(disableChildren) => (
                  <Button type="submit" disabled={disabled || disableChildren}>
                    Done
                  </Button>
                )}
              />
            </>
          }
        />
      </form>
    </Modal>
  );
};

export default EditTargetsModal;
