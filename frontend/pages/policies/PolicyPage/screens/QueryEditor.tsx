import React, { useContext, useEffect } from "react";
import { Link } from "react-router";
import { useDispatch } from "react-redux";
import { InjectedRouter } from "react-router/lib/Router";

import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";
import { AppContext } from "context/app";
import { PolicyContext } from "context/policy"; // @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import PATHS from "router/paths"; // @ts-ignore
import debounce from "utilities/debounce"; // @ts-ignore
import deepDifference from "utilities/deep_difference";
import { IPolicyFormData, IPolicy } from "interfaces/policy";

import PolicyForm from "pages/policies/PolicyPage/components/PolicyForm";
import BackChevron from "../../../../../assets/images/icon-chevron-down-9x6@2x.png";

interface IQueryEditorProps {
  router: InjectedRouter;
  baseClass: string;
  policyIdForEdit: number | null;
  storedPolicy: IPolicy | undefined;
  storedPolicyError: any;
  showOpenSchemaActionText: boolean;
  isStoredPolicyLoading: boolean;
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
  createPolicy,
  onOsqueryTableSelect,
  goToSelectTargets,
  onOpenSchemaSidebar,
  renderLiveQueryWarning,
}: IQueryEditorProps): JSX.Element | null => {
  const dispatch = useDispatch();
  const { currentUser } = useContext(AppContext);

  // Note: The PolicyContext values should always be used for any mutable policy data such as query name
  // The storedPolicy prop should only be used to access immutable metadata such as author id
  const {
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryBody,
    policyTeamId,
  } = useContext(PolicyContext);

  useEffect(() => {
    if (storedPolicyError) {
      dispatch(
        renderFlash(
          "error",
          "Something went wrong retrieving your policy. Please try again."
        )
      );
    }
  }, []);

  const onCreatePolicy = debounce(async (formData: IPolicyFormData) => {
    // TODO: The approach taken with selectedTeamId context works in most cases. Howeve, the context
    // will reset to global if page is refreshed. This will cause bugs where a global policy gets
    // created when the user intended a team policy. For non-gloabl users, request will fail but the
    // erorr is opaque and would require them to navigate back to the manage policies page to select
    // a team and start over, in which case it might be better to intercept the unauthorized errors
    // and redirect to the manage policies page (unless we have added a means to select a team on
    // the edit/create policy form itself).
    if (policyTeamId) {
      formData.team_id = policyTeamId;
    }

    try {
      const policy: IPolicy = await createPolicy(formData).then(
        (data) => data.policy
      );
      router.push(PATHS.EDIT_POLICY(policy));
      dispatch(renderFlash("success", "Policy created!"));
    } catch (createError) {
      console.error(createError);
      dispatch(
        renderFlash(
          "error",
          "Something went wrong creating your policy. Please try again."
        )
      );
    }
  });

  const onUpdatePolicy = async (formData: IPolicyFormData) => {
    if (!policyIdForEdit) {
      return false;
    }

    const updatedPolicy = deepDifference(formData, {
      lastEditedQueryName,
      lastEditedQueryDescription,
      lastEditedQueryBody,
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
      dispatch(renderFlash("success", "Policy updated!"));
    } catch (updateError) {
      console.error(updateError);
      dispatch(
        renderFlash(
          "error",
          "Something went wrong updating your policy. Please try again."
        )
      );
    }

    return false;
  };

  if (!currentUser) {
    return null;
  }

  const backPath = policyTeamId ? `?team_id=${policyTeamId}` : "";

  return (
    <div className={`${baseClass}__form body-wrap`}>
      <Link
        to={`${PATHS.MANAGE_POLICIES}/${backPath}`}
        className={`${baseClass}__back-link`}
      >
        <img src={BackChevron} alt="back chevron" id="back-chevron" />
        <span>Back to policies</span>
      </Link>
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
      />
    </div>
  );
};

export default QueryEditor;
