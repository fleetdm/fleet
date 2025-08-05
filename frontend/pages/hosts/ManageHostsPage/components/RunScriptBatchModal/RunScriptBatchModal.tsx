import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";

import classnames from "classnames";

import Radio from "components/forms/fields/Radio";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import TooltipWrapper from "components/TooltipWrapper";

import { NotificationContext } from "context/notification";

import { addTeamIdCriteria, IScript } from "interfaces/script";
import { getErrorReason } from "interfaces/errors";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Modal from "components/Modal";

import scriptsAPI, {
  IListScriptsQueryKey,
  IScriptBatchSupportedFilters,
  IScriptsResponse,
  IRunScriptBatchRequest,
} from "services/entities/scripts";
import ScriptDetailsModal from "pages/hosts/components/ScriptDetailsModal";
import Spinner from "components/Spinner";
import EmptyTable from "components/EmptyTable";
import Button from "components/buttons/Button";

import RunScriptBatchPaginatedList from "../RunScriptBatchPaginatedList";
import { IPaginatedListScript } from "../RunScriptBatchPaginatedList/RunScriptBatchPaginatedList";
import {
  validateFormData,
  IRunScriptBatchModalFormValidation,
} from "./helpers";

const baseClass = "run-script-batch-modal";

export interface IRunScriptBatchModalScheduleFormData {
  date: string;
  time: string;
}
interface IRunScriptBatchModal {
  runByFilters: boolean; // otherwise, by selectedHostIds
  // since teamId has multiple uses in this component, it's passed in as its own prop and added to
  // `filters` as needed
  filters: Omit<IScriptBatchSupportedFilters, "team_id">;
  teamId: number;
  // If we are on the free tier, we don't want to apply any kind of team filters (since the feature is Premium only).
  isFreeTier?: boolean;
  totalFilteredHostsCount: number;
  selectedHostIds: number[];
  onCancel: () => void;
}

const RunScriptBatchModal = ({
  runByFilters = false,
  filters,
  totalFilteredHostsCount,
  selectedHostIds,
  teamId,
  isFreeTier,
  onCancel,
}: IRunScriptBatchModal) => {
  const { renderFlash } = useContext(NotificationContext);

  const [batchRunDate, setBatchRunDate] = useState<string>("");
  const [batchRunTime, setBatchRunTime] = useState<string>("");
  const [
    formValidation,
    setFormValidation,
  ] = useState<IRunScriptBatchModalFormValidation>(() =>
    validateFormData({ date: batchRunDate, time: batchRunTime })
  );

  const [runMode, setRunMode] = useState<"run_now" | "schedule">("run_now");
  const [selectedScript, setSelectedScript] = useState<IScript | undefined>(
    undefined
  );
  const [isUpdating, setIsUpdating] = useState(false);
  const [scriptForDetails, setScriptForDetails] = useState<
    IPaginatedListScript | undefined
  >(undefined);
  // just used to get the total number of scripts, could be optimized by implementing a dedicated scriptsCount endpoint
  const { data: scripts } = useQuery<
    IScriptsResponse,
    Error,
    IScript[],
    IListScriptsQueryKey[]
  >(
    [addTeamIdCriteria({ scope: "scripts" }, teamId, isFreeTier)],
    ({ queryKey }) => {
      return scriptsAPI.getScripts(queryKey[0]);
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      keepPreviousData: true,
      select: (data) => {
        return data.scripts || [];
      },
    }
  );

  const onChangeRunMode = (mode: "run_now" | "schedule") => {
    setRunMode(mode);
    setFormValidation(
      validateFormData({ date: batchRunDate, time: batchRunTime }, mode)
    );
  };

  const onInputChange = (update: { name: string; value: string }) => {
    if (update.name === "date") {
      setBatchRunDate(update.value);
    } else if (update.name === "time") {
      setBatchRunTime(update.value);
    }
    setFormValidation(
      validateFormData(
        {
          date: batchRunDate,
          time: batchRunTime,
          [update.name]: update.value,
        },
        runMode
      )
    );
  };

  const onRunScriptBatch = useCallback(
    async (script: IScript) => {
      setIsUpdating(true);

      // Create the base request.
      let body: IRunScriptBatchRequest;

      if (runByFilters) {
        body = {
          script_id: script.id,
          filters: addTeamIdCriteria(filters, teamId, isFreeTier),
        };
      } else {
        body = {
          script_id: script.id,
          host_ids: selectedHostIds,
        };
      }

      // Add not_before if scheduling
      if (runMode === "schedule") {
        body.not_before = `${batchRunDate} ${batchRunTime}:00.000Z`;
      }

      try {
        await scriptsAPI.runScriptBatch(body);
        renderFlash(
          "success",
          `Script is running on ${
            runByFilters
              ? totalFilteredHostsCount.toLocaleString()
              : selectedHostIds.length.toLocaleString()
          } hosts, or will run as each host comes online. See host details for individual results.`
        );
      } catch (error) {
        let errorMessage = "Could not run script.";
        if (getErrorReason(error).includes("too many hosts")) {
          errorMessage =
            "Could not run script: too many hosts targeted. Please try again with fewer hosts.";
        }
        renderFlash("error", errorMessage);
        // can determine more specific error case with additional call to upcoming summary endpoint
      } finally {
        setIsUpdating(false);
      }
    },
    [renderFlash, selectedHostIds, runMode, batchRunDate, batchRunTime]
  );

  const renderModalContent = () => {
    if (scripts === undefined) {
      return <Spinner />;
    }
    if (!scripts.length) {
      return (
        <EmptyTable
          header="No scripts available for this team"
          info={
            <>
              You can add saved scripts{" "}
              <a
                href={
                  isFreeTier
                    ? "/controls/scripts"
                    : `/controls/scripts?team_id=${teamId}`
                }
              >
                here
              </a>
              .
            </>
          }
        />
      );
    }
    if (!selectedScript) {
      const targetCount = runByFilters
        ? totalFilteredHostsCount
        : selectedHostIds.length;
      return (
        <>
          <p>
            Will run on{" "}
            <b>
              {targetCount.toLocaleString()} host{targetCount > 1 ? "s" : ""}
            </b>
            . You can see individual script results on the host details page.
          </p>
          <RunScriptBatchPaginatedList
            onRunScript={(script) => setSelectedScript(script)}
            isUpdating={isUpdating}
            teamId={teamId}
            isFreeTier={isFreeTier}
            scriptCount={scripts.length}
            setScriptForDetails={setScriptForDetails}
          />
        </>
      );
    }
    const platforms =
      selectedScript.name.indexOf(".ps1") > 0 ? "windows" : "macOS/linux";
    return (
      <div className={`${baseClass}__script-schedule`}>
        <p>
          <b>{selectedScript.name}</b> will run on compatible hosts ({platforms}
          ).
        </p>
        <div className={`${baseClass}__script-run-mode-form`}>
          <div className="form-field">
            <div className="form-field__label">Schedule</div>
            <Radio
              className={`${baseClass}__radio-input`}
              label="Run now"
              id="run-now-batch-scripts-radio-btn"
              checked={runMode === "run_now"}
              value="Run now"
              name="run-mode"
              onChange={() => onChangeRunMode("run_now")}
            />
            <Radio
              className={`${baseClass}__radio-input`}
              label="Schedule for later"
              id="custom-target-radio-btn"
              checked={runMode === "schedule"}
              value="Custom"
              name="target-type"
              onChange={() => onChangeRunMode("schedule")}
            />
          </div>
          {runMode === "schedule" && (
            <div className={`${baseClass}__script-schedule-form`}>
              <span className="date-time-inputs">
                <InputField
                  onChange={onInputChange}
                  value={batchRunDate}
                  label="Date (UTC)"
                  name="date"
                  parseTarget
                  helpText='YYYY-MM-DD format (e.g., "2024-07-01").'
                  error={formValidation.date?.message}
                />
                <InputField
                  onChange={onInputChange}
                  value={batchRunTime}
                  label="Time (UTC)"
                  name="time"
                  parseTarget
                  helpText='HH:MM 24-hour format (e.g., "13:37").'
                  error={formValidation.time?.message}
                />
              </span>
            </div>
          )}
        </div>
      </div>
    );
  };

  const classes = classnames(baseClass, {
    [`${baseClass}__hide-main`]: !!scriptForDetails,
  });

  return (
    <>
      <Modal
        title="Run script"
        onExit={onCancel}
        onEnter={onCancel}
        className={classes}
        disableClosingModal={isUpdating}
      >
        <>
          {renderModalContent()}
          {!selectedScript && !scriptForDetails && (
            <div className="modal-cta-wrap">
              <Button disabled={isUpdating} onClick={onCancel}>
                Done
              </Button>
            </div>
          )}
          {selectedScript && (
            <div className="modal-cta-wrap">
              <TooltipWrapper
                tipContent="Enter a date and time to schedule this script."
                underline={false}
                position="top"
                disableTooltip={formValidation.isValid}
                showArrow
              >
                <Button
                  disabled={isUpdating || !formValidation.isValid}
                  onClick={() => onRunScriptBatch(selectedScript)}
                  isLoading={isUpdating}
                >
                  Run
                </Button>
              </TooltipWrapper>
              <Button
                disabled={isUpdating}
                variant="inverse"
                onClick={() => {
                  setSelectedScript(undefined);
                }}
              >
                Go back
              </Button>
              <Button
                disabled={isUpdating}
                variant="inverse"
                onClick={onCancel}
              >
                Cancel
              </Button>
            </div>
          )}
        </>
      </Modal>
      {!!scriptForDetails && !selectedScript && (
        <ScriptDetailsModal
          onCancel={() => setScriptForDetails(undefined)}
          selectedScriptDetails={scriptForDetails}
          suppressSecondaryActions
          customPrimaryButtons={
            <div className="modal-cta-wrap">
              <Button
                onClick={() => {
                  setScriptForDetails(undefined);
                  setSelectedScript(scriptForDetails);
                }}
                isLoading={isUpdating}
              >
                Run
              </Button>
              <Button
                onClick={() => setScriptForDetails(undefined)}
                variant="inverse"
              >
                Go back
              </Button>
            </div>
          }
        />
      )}
    </>
  );
};

export default RunScriptBatchModal;
