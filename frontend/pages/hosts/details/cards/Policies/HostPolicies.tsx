import React, { useCallback } from "react";

import { Row } from "react-table";

import { isAndroid } from "interfaces/platform";
import { IHostPolicy } from "interfaces/policy";
import { SUPPORT_LINK } from "utilities/constants";
import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import EmptyState from "components/EmptyState";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import InfoBanner from "components/InfoBanner";
import IconStatusMessage from "components/IconStatusMessage";

import {
  generatePolicyTableHeaders,
  generatePolicyDataSet,
} from "./HostPoliciesTable/HostPoliciesTableConfig";
import PolicyFailingCount from "./HostPoliciesTable/PolicyFailingCount";

const baseClass = "host-policies-card";

interface IPoliciesProps {
  policies: IHostPolicy[];
  isLoading: boolean;
  deviceUser?: boolean;
  togglePolicyDetailsModal: (policy: IHostPolicy) => void;
  hostPlatform: string;

  currentTeamId?: number;
  conditionalAccessEnabled?: boolean;
  conditionalAccessBypassed?: boolean;
  canManagePolicies?: boolean;
  onManagePolicies?: () => void;
}

interface IHostPoliciesRowProps extends Row {
  original: IHostPolicy;
}

const Policies = ({
  policies,
  isLoading,
  deviceUser,
  togglePolicyDetailsModal,
  hostPlatform,

  currentTeamId,
  conditionalAccessEnabled,
  conditionalAccessBypassed,
  canManagePolicies,
  onManagePolicies,
}: IPoliciesProps): JSX.Element => {
  const tableHeaders = generatePolicyTableHeaders(currentTeamId);
  if (deviceUser) {
    // Remove view all hosts link
    tableHeaders.pop();
  }
  const failingResponses: IHostPolicy[] =
    policies.filter((policy: IHostPolicy) => policy.response === "fail") || [];

  const onClickRow = useCallback(
    (row: IHostPoliciesRowProps) => {
      togglePolicyDetailsModal(row.original);
    },
    [togglePolicyDetailsModal]
  );

  const renderBanner = () => {
    if (!failingResponses?.length) {
      return null;
    }
    if (conditionalAccessBypassed) {
      return (
        <InfoBanner borderRadius="xlarge">
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

    if (policies.length === 0) {
      const target = deviceUser ? "your device" : "this host";
      const manageClause = canManagePolicies
        ? ", or manage its policies."
        : ".";

      return (
        <>
          <TableCount name="policies" count={0} />
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
        </>
      );
    }

    return (
      <>
        {renderBanner()}
        <TableContainer
          columnConfigs={tableHeaders}
          data={generatePolicyDataSet(policies, !!conditionalAccessEnabled)}
          isLoading={isLoading}
          defaultSortHeader="status"
          resultsTitle="policies"
          emptyComponent={() => <></>}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          renderCount={() => (
            <TableCount name="policies" count={policies.length} />
          )}
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
