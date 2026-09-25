import React, { useCallback, useEffect, useMemo } from "react";
import { Row } from "react-table";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import EmptyState from "components/EmptyState";
import Slider from "components/forms/fields/Slider";
import IconStatusMessage from "components/IconStatusMessage";
import InfoBanner from "components/InfoBanner";
import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import { isAndroid } from "interfaces/platform";
import { IHostPolicy } from "interfaces/policy";
import { SUPPORT_LINK } from "utilities/constants";

import {
  generatePolicyTableHeaders,
  generatePolicyDataSet,
} from "./HostPoliciesTable/HostPoliciesTableConfig";
import PolicyFailingCount from "./HostPoliciesTable/PolicyFailingCount";

const baseClass = "host-policies-card";

interface IPoliciesBaseProps {
  policies: IHostPolicy[];
  isLoading: boolean;
  togglePolicyDetailsModal: (policy: IHostPolicy) => void;
  closePolicyDetailsModal: () => void;
  hostPlatform: string;
  conditionalAccessEnabled?: boolean;
  conditionalAccessBypassed?: boolean;
}

interface IHostDetailsPoliciesProps extends IPoliciesBaseProps {
  deviceUser?: false;
  currentTeamId?: number;
  canManagePolicies?: boolean;
  onManagePolicies?: () => void;
}

interface IMyDevicePoliciesProps extends IPoliciesBaseProps {
  deviceUser: true;
  /** Hidden policies are excluded from `policies` until toggled on. */
  showHiddenPolicies: boolean;
  onToggleShowHiddenPolicies: () => void;
}

type IPoliciesProps = IHostDetailsPoliciesProps | IMyDevicePoliciesProps;

const HIDDEN_POLICIES_TOOLTIP =
  "Some policies running on this device have been hidden by your IT admin. These include auto-remediated compliance checks, and other issues that don't require action from you.";

interface IHostPoliciesRowProps extends Row {
  original: IHostPolicy;
}

const Policies = (props: IPoliciesProps): JSX.Element => {
  const {
    policies,
    isLoading,
    deviceUser,
    togglePolicyDetailsModal,
    closePolicyDetailsModal,
    hostPlatform,
    conditionalAccessEnabled,
    conditionalAccessBypassed,
  } = props;
  const currentTeamId = props.deviceUser ? undefined : props.currentTeamId;
  const canManagePolicies = !props.deviceUser && !!props.canManagePolicies;
  const onManagePolicies = props.deviceUser
    ? undefined
    : props.onManagePolicies;

  const tableHeaders = generatePolicyTableHeaders(currentTeamId);
  if (deviceUser) {
    // Remove view all hosts link
    tableHeaders.pop();
  }
  const failingResponses: IHostPolicy[] =
    policies.filter((policy: IHostPolicy) => policy.response === "fail") || [];

  useEffect(() => {
    return () => {
      closePolicyDetailsModal();
    };
  }, [closePolicyDetailsModal]);

  const onClickRow = useCallback(
    (row: IHostPoliciesRowProps) => {
      togglePolicyDetailsModal(row.original);
    },
    [togglePolicyDetailsModal]
  );

  // Memoize the table data so its reference stays stable across re-renders
  // that don't change the policies.
  const tableData = useMemo(
    () => generatePolicyDataSet(policies, !!conditionalAccessEnabled),
    [policies, conditionalAccessEnabled]
  );

  const renderBanner = () => {
    if (!failingResponses?.length) {
      return null;
    }
    if (conditionalAccessBypassed) {
      return (
        <InfoBanner>
          <IconStatusMessage
            iconName="clock"
            iconColor="ui-fleet-black-50"
            message={
              <span>
                <strong>Access restored for next Okta login</strong>
                <br />
                To fully restore access, click on the policies marked
                &apos;Action required&apos; and follow the resolution steps.
                Once resolved, click &apos;Refetch&apos; to check status.
              </span>
            }
          />
        </InfoBanner>
      );
    }
    return (
      <PolicyFailingCount
        policyList={policies}
        deviceUser={deviceUser}
        conditionalAccessEnabled={conditionalAccessEnabled}
      />
    );
  };

  const renderShowHiddenToggle = () => {
    if (!props.deviceUser) {
      return null;
    }
    return (
      <Slider
        className={`${baseClass}__show-hidden-toggle`}
        value={props.showHiddenPolicies}
        onChange={props.onToggleShowHiddenPolicies}
        activeText="Show hidden policies"
        inactiveText="Show hidden policies"
        ariaLabel="Show hidden policies"
        labelTooltip={HIDDEN_POLICIES_TOOLTIP}
      />
    );
  };

  const renderHostPolicies = () => {
    if (hostPlatform === "ios" || hostPlatform === "ipados") {
      return (
        <EmptyState
          header={<>Policies are not supported for this host</>}
          info={
            <>
              Interested in detecting device health issues on{" "}
              {hostPlatform === "ios" ? "iPhones" : "iPads"}?{" "}
              <CustomLink url={SUPPORT_LINK} text="Let us know" newTab />
            </>
          }
        />
      );
    }

    if (isAndroid(hostPlatform)) {
      return (
        <EmptyState
          header={<>Policies are not supported for this host</>}
          info={
            <>
              Interested in detecting device health issues on Android hosts?{" "}
              <CustomLink url={SUPPORT_LINK} text="Let us know" newTab />
            </>
          }
        />
      );
    }

    const target = deviceUser ? "your device" : "this host";
    const manageClause = canManagePolicies ? ", or manage its policies." : ".";

    return (
      <>
        {renderBanner()}
        <TableContainer
          columnConfigs={tableHeaders}
          data={tableData}
          isLoading={isLoading}
          defaultSortHeader="status"
          resultsTitle="policies"
          emptyComponent={() => (
            <EmptyState
              header="No policies checked"
              info={`Select Refetch to load the latest data from ${target}${manageClause}`}
              primaryButton={
                canManagePolicies ? (
                  <Button onClick={onManagePolicies} type="button">
                    Manage policies
                  </Button>
                ) : undefined
              }
            />
          )}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          renderCount={() => (
            <TableCount name="policies" count={policies.length} />
          )}
          customControl={renderShowHiddenToggle}
          disableMultiRowSelect // Removes hover/click state
          isClientSidePagination
          onClickRow={onClickRow}
          keyboardSelectableRows
        />
      </>
    );
  };

  return <div className={baseClass}>{renderHostPolicies()}</div>;
};

export default Policies;
