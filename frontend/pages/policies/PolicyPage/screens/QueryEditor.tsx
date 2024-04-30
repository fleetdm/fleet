import React, { useContext, useEffect, useState } from "react";
import { InjectedRouter } from "react-router/lib/Router";

import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";
import aiAutofillAPI from "services/entities/ai_autofill";
import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { NotificationContext } from "context/notification";
import PATHS from "router/paths";
import debounce from "utilities/debounce";
import deepDifference from "utilities/deep_difference";
import { IPolicyFormData, IPolicy } from "interfaces/policy";

import BackLink from "components/BackLink";
import PolicyForm from "pages/policies/PolicyPage/components/PolicyForm";

interface IQueryEditorProps {
  router: InjectedRouter;
  baseClass: string;
  policyIdForEdit: number | null;
  storedPolicy: IPolicy | undefined;
  storedPolicyError: Error | null;
  showOpenSchemaActionText: boolean;
  isStoredPolicyLoading: boolean;
  isTeamAdmin: boolean;
  isTeamMaintainer: boolean;
  isTeamObserver: boolean;
  createPolicy: (formData: IPolicyFormData) => Promise<any>;
  onOsqueryTableSelect: (tableName: string) => void;
  goToSelectTargets: () => void;
  onOpenSchemaSidebar: () => void;
  renderLiveQueryWarning: () => JSX.Element | null;
}

const QueryEditor = ({
  router,
  baseClass,
  policyIdForEdit,
  storedPolicy,
  storedPolicyError,
  showOpenSchemaActionText,
  isStoredPolicyLoading,
  isTeamAdmin,
  isTeamMaintainer,
  isTeamObserver,
  createPolicy,
  onOsqueryTableSelect,
  goToSelectTargets,
  onOpenSchemaSidebar,
  renderLiveQueryWarning,
}: IQueryEditorProps): JSX.Element | null => {
  const { currentUser, isPremiumTier, filteredPoliciesPath } = useContext(
    AppContext
  );
  const { renderFlash } = useContext(NotificationContext);

  // Note: The PolicyContext values should always be used for any mutable policy data such as query name
  // The storedPolicy prop should only be used to access immutable metadata such as author id
  const {
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryBody,
    lastEditedQueryResolution,
    lastEditedQueryCritical,
    lastEditedQueryPlatform,
    policyTeamId,
    setLastEditedQueryDescription,
    setLastEditedQueryResolution,
  } = useContext(PolicyContext);

  useEffect(() => {
    if (storedPolicyError) {
      renderFlash(
        "error",
        "Something went wrong retrieving your policy. Please try again."
      );
    }
  }, []);

  const [isUpdatingPolicy, setIsUpdatingPolicy] = useState(false);
  const [backendValidators, setBackendValidators] = useState<{
    [key: string]: string;
  }>({});
  const [aiAutofillData, setAiAutofillData] = useState<{
    description: string;
    resolution: string;
  } | null>(null);
  const [aiAutofillErrors, setAiAutofillErrors] = useState<any>({});
  const [isFetchingAIAutofill, setIsFetchingAiAutofill] = useState({
    description: false,
    resolution: false,
  });

  const onAiAutofill = async (fetching: {
    description: boolean;
    resolution: boolean;
  }) => {
    // When AI autofill data exists already, fill out section clicked with data
    if (aiAutofillData) {
      if (fetching.description) {
        setLastEditedQueryDescription(aiAutofillData.description);
      }
      if (fetching.resolution) {
        setLastEditedQueryResolution(aiAutofillData.resolution);
      }
    } else {
      // Show thinking state and fetch data from API
      setIsFetchingAiAutofill(fetching);

      try {
        const autofillResponse = await aiAutofillAPI.getHumanInterpretationFromSQL(
          lastEditedQueryBody
        );

        setAiAutofillData(autofillResponse);

        // Only fill out section that was clicked to be fetched
        if (fetching.description) {
          setLastEditedQueryDescription(autofillResponse.description);
        }
        if (fetching.resolution) {
          setLastEditedQueryResolution(autofillResponse.resolution);
        }
      } catch (error) {
        console.log(error);
        renderFlash("error", "Couldn't autofill policy data.");
        setAiAutofillErrors(error);
      }
      setIsFetchingAiAutofill({ description: false, resolution: false });
    }
  };

  const onCreatePolicy = debounce(async (formData: IPolicyFormData) => {
    if (policyTeamId) {
      formData.team_id = policyTeamId;
    }
    setIsUpdatingPolicy(true);
    const payload: IPolicyFormData = {
      name: formData.name,
      description: formData.description,
      query: formData.query,
      resolution: formData.resolution,
      platform: formData.platform,
    };
    if (isPremiumTier) {
      payload.critical = formData.critical;
      payload.team_id = formData.team_id;
    }

    try {
      const policy: IPolicy = await createPolicy(payload).then(
        (data) => data.policy
      );
      setIsUpdatingPolicy(false);
      router.push(PATHS.EDIT_POLICY(policy));
      renderFlash("success", "Policy created!");
    } catch (createError: any) {
      console.error(createError);
      if (createError.data.errors[0].reason.includes("already exists")) {
        setBackendValidators({
          name: "A policy with this name already exists",
        });
      } else {
        renderFlash(
          "error",
          "Something went wrong creating your policy. Please try again."
        );
      }
    } finally {
      setIsUpdatingPolicy(false);
    }
  });

  const onUpdatePolicy = async (formData: IPolicyFormData) => {
    if (!policyIdForEdit) {
      return false;
    }

    setIsUpdatingPolicy(true);

    const updatedPolicy = deepDifference(formData, {
      lastEditedQueryName,
      lastEditedQueryDescription,
      lastEditedQueryBody,
      lastEditedQueryResolution,
      lastEditedQueryCritical,
      lastEditedQueryPlatform,
    });

    const updateAPIRequest = () => {
      // storedPolicy.team_id is used for existing policies because selectedTeamId is subject to change
      const team_id = storedPolicy?.team_id;

      return team_id
        ? teamPoliciesAPI.update(policyIdForEdit, {
            ...updatedPolicy,
            team_id,
          })
        : globalPoliciesAPI.update(policyIdForEdit, updatedPolicy);
    };

    try {
      await updateAPIRequest();
      renderFlash("success", "Policy updated!");
    } catch (updateError: any) {
      console.error(updateError);
      if (updateError.data.errors[0].reason.includes("Duplicate")) {
        renderFlash("error", "A policy with this name already exists.");
      } else {
        renderFlash(
          "error",
          "Something went wrong updating your policy. Please try again."
        );
      }
    } finally {
      setIsUpdatingPolicy(false);
    }

    return false;
  };

  if (!currentUser) {
    return null;
  }

  // Function instead of constant eliminates race condition with filteredPoliciesPath
  const backToPoliciesPath = () => {
    return filteredPoliciesPath || PATHS.MANAGE_POLICIES;
  };

  return (
    <div className={`${baseClass}__form`}>
      <div className={`${baseClass}__header-links`}>
        <BackLink text="Back to policies" path={backToPoliciesPath()} />
      </div>
      <PolicyForm
        onCreatePolicy={onCreatePolicy}
        goToSelectTargets={goToSelectTargets}
        onOsqueryTableSelect={onOsqueryTableSelect}
        onUpdate={onUpdatePolicy}
        storedPolicy={storedPolicy}
        policyIdForEdit={policyIdForEdit}
        isStoredPolicyLoading={isStoredPolicyLoading}
        showOpenSchemaActionText={showOpenSchemaActionText}
        onOpenSchemaSidebar={onOpenSchemaSidebar}
        renderLiveQueryWarning={renderLiveQueryWarning}
        backendValidators={backendValidators}
        isTeamAdmin={isTeamAdmin}
        isTeamMaintainer={isTeamMaintainer}
        isTeamObserver={isTeamObserver}
        isUpdatingPolicy={isUpdatingPolicy}
        isFetchingAIAutofill={isFetchingAIAutofill}
        onAiAutofill={onAiAutofill}
        resetAiAutofillData={() => setAiAutofillData(null)}
      />
    </div>
  );
};

export default QueryEditor;
