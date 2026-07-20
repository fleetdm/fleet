import React, { useRef } from "react";

import { notify } from "components/ToastNotification";
import { IPolicyStats } from "interfaces/policy";
import { IConfig } from "interfaces/config";
import { ITeamConfig } from "interfaces/team";
import { PLATFORM_DISPLAY_NAMES, QueryablePlatform } from "interfaces/platform";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import PolicyAutomationsFields, {
  IPolicyAutomationsFieldsHandle,
} from "pages/policies/components/PolicyAutomationsFields";
import { useUpdatePolicyAutomations } from "pages/policies/hooks";

const baseClass = "manage-automations-modal";

const PLATFORM_DISPLAY_ORDER: QueryablePlatform[] = [
  "darwin",
  "windows",
  "linux",
  "chrome",
];

const SUCCESS_MSG = "Successfully updated policy automations.";
const ERR_MSG = "Could not update policy automations.";

interface IManageAutomationsModalProps {
  policy: IPolicyStats;
  fleetName: string;
  isGlobalPolicy: boolean;
  /** undefined for "All fleets", 0 for "Unassigned", positive for a fleet. */
  teamIdForApi: number | undefined;
  automationsConfig: IConfig | ITeamConfig | undefined;
  globalConfig: IConfig | undefined;
  refetchPolicies: () => void;
  onExit: () => void;
}

const ManageAutomationsModal = ({
  policy,
  fleetName,
  isGlobalPolicy,
  teamIdForApi,
  automationsConfig,
  globalConfig,
  refetchPolicies,
  onExit,
}: IManageAutomationsModalProps): JSX.Element => {
  const automationsRef = useRef<IPolicyAutomationsFieldsHandle>(null);

  const { mutate: save, isLoading: isSaving } = useUpdatePolicyAutomations({
    policy,
    teamIdForApi,
    isGlobalPolicy,
    automationsConfig,
    onSuccess: () => {
      notify.success(SUCCESS_MSG);
      refetchPolicies();
      onExit();
    },
    onError: () => notify.error(ERR_MSG),
  });

  const handleSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    const payload = automationsRef.current?.getAutomationsPayload();
    if (!payload) {
      return;
    }
    if (!payload.isValid) {
      return;
    }
    if (!payload.isDirty) {
      onExit();
      return;
    }
    save({
      policyUpdate: payload.policyUpdate,
      webhookOrTicketUpdate: payload.webhookOrTicketUpdate,
    });
  };

  const policyPlatforms = (policy.platform ?? "")
    .split(",")
    .map((p) => p.trim())
    .filter((p): p is QueryablePlatform =>
      (PLATFORM_DISPLAY_ORDER as string[]).includes(p)
    );
  const displayedPlatforms = PLATFORM_DISPLAY_ORDER.filter((p) =>
    policyPlatforms.includes(p)
  );

  return (
    <Modal
      title="Manage automations"
      onExit={onExit}
      className={baseClass}
      width="large"
      isContentDisabled={isSaving}
    >
      <form onSubmit={handleSubmit}>
        <div className={`${baseClass}__body`}>
          <div className={`${baseClass}__header`}>
            Manage automations for the <b>{policy.name}</b> policy on{" "}
            <b>{fleetName}</b>.
          </div>

          {displayedPlatforms.length > 0 && (
            <section className={`${baseClass}__section`}>
              <h2 className={`${baseClass}__section-title`}>Platforms</h2>
              <div className={`${baseClass}__platforms`}>
                {displayedPlatforms.map((p) => (
                  <span key={p} className={`${baseClass}__platform`}>
                    <Icon name={p} size="small" />
                    {PLATFORM_DISPLAY_NAMES[p]}
                  </span>
                ))}
              </div>
            </section>
          )}

          <section className={`${baseClass}__section`}>
            <h2 className={`${baseClass}__section-title`}>Automations</h2>
            <PolicyAutomationsFields
              ref={automationsRef}
              policy={policy}
              isGlobalPolicy={isGlobalPolicy}
              teamIdForApi={teamIdForApi}
              automationsConfig={automationsConfig}
              globalConfig={globalConfig}
              fleetName={fleetName}
            />
          </section>
        </div>

        <div className="modal-cta-wrap">
          <Button type="submit" isLoading={isSaving} disabled={isSaving}>
            Save
          </Button>
          <Button type="button" onClick={onExit} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default ManageAutomationsModal;
