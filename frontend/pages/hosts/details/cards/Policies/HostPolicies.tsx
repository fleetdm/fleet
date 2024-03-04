import React from "react";

import { IHostPolicy } from "interfaces/policy";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import Card from "components/Card";

import {
  generatePolicyTableHeaders,
  generatePolicyDataSet,
} from "./HostPoliciesTable/HostPoliciesTableConfig";
import PolicyFailingCount from "./HostPoliciesTable/PolicyFailingCount";

const baseClass = "policies-card";

interface IPoliciesProps {
  policies: IHostPolicy[];
  isLoading: boolean;
  deviceUser?: boolean;
  togglePolicyDetailsModal: (policy: IHostPolicy) => void;
}

const Policies = ({
  policies,
  isLoading,
  deviceUser,
  togglePolicyDetailsModal,
}: IPoliciesProps): JSX.Element => {
  if (policies.length === 0) {
    return (
      <Card
        borderRadiusSize="large"
        includeShadow
        largePadding
        className={baseClass}
      >
        <p className="card__header">Policies</p>
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
      </Card>
    );
  }

  const tableHeaders = generatePolicyTableHeaders(togglePolicyDetailsModal);
  if (deviceUser) {
    // Remove view all hosts link
    tableHeaders.pop();
  }
  const failingResponses: IHostPolicy[] =
    policies.filter((policy: IHostPolicy) => policy.response === "fail") || [];

  return (
    <Card
      borderRadiusSize="large"
      includeShadow
      largePadding
      className={baseClass}
    >
      <p className="card__header">Policies</p>

      {policies.length > 0 && (
        <>
          {failingResponses?.length > 0 && (
            <PolicyFailingCount policyList={policies} deviceUser={deviceUser} />
          )}
          <TableContainer
            columnConfigs={tableHeaders}
            data={generatePolicyDataSet(policies)}
            isLoading={isLoading}
            manualSortBy
            resultsTitle="policy items"
            emptyComponent={() => <></>}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            disablePagination
            disableCount
            disableMultiRowSelect
          />
        </>
      )}
    </Card>
  );
};

export default Policies;
