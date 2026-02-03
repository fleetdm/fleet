import React, { useCallback } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import { isAndroid } from "interfaces/platform";
import { IHostPolicy } from "interfaces/policy";
import { SUPPORT_LINK } from "utilities/constants";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import CardHeader from "components/CardHeader";
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
  router: InjectedRouter;
  currentTeamId?: number;
  conditionalAccessEnabled?: boolean;
  conditionalAccessBypassed?: boolean;
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
  router,
  currentTeamId,
  conditionalAccessEnabled,
  conditionalAccessBypassed,
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
    [router]
  );

  const renderHostPolicies = () => {
    if (hostPlatform === "ios" || hostPlatform === "ipados") {
      return (
        <EmptyTable
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
        <EmptyTable
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
      return (
        <EmptyTable
          header={
            <>
              No policies are checked{" "}
              {deviceUser ? `on your device` : `for this host`}
            </>
          }
          info={
            <>
              Expecting to see policies? Try selecting “Refetch” to ask{" "}
              {deviceUser ? `your device ` : `this host `}
              to report new vitals.
            </>
          }
        />
      );
    }

    return (
      <>
        {failingResponses?.length > 0 && !conditionalAccessBypassed ? (
          <PolicyFailingCount
            policyList={policies}
            deviceUser={deviceUser}
            conditionalAccessEnabled={conditionalAccessEnabled}
          />
        ) : (
          <InfoBanner color="grey" borderRadius="xlarge">
            <IconStatusMessage
              iconName="clock"
              iconColor="ui-fleet-black-50"
              message={
                <span>
                  <strong>Access restored for next Okta login</strong>
                  <br />
                  {`To fully restore access, click on the policies marked "Action
                  required" and follow the resolution steps. Once resolved,
                  click "Refetch" to check status.`}
                </span>
              }
            />
          </InfoBanner>
        )}
        <TableContainer
          columnConfigs={tableHeaders}
          data={generatePolicyDataSet(policies, !!conditionalAccessEnabled)}
          isLoading={isLoading}
          defaultSortHeader="status"
          resultsTitle="policies"
          emptyComponent={() => <></>}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          disableCount
          disableMultiRowSelect // Removes hover/click state
          isClientSidePagination
          onClickRow={onClickRow}
          keyboardSelectableRows
        />
      </>
    );
  };

  return (
    <div className={baseClass}>
      <CardHeader header="Policies" />
      {renderHostPolicies()}
    </div>
  );
};

export default Policies;
