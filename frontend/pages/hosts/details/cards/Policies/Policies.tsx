import React from "react";

import { IHostPolicy } from "interfaces/policy";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";

import {
  generatePolicyTableHeaders,
  generatePolicyDataSet,
} from "./HostPoliciesTable/HostPoliciesTableConfig";
import PolicyFailingCount from "./HostPoliciesTable/PolicyFailingCount";

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
      <div className="section section--policies">
        <p className="section__header">Policies</p>
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
      </div>
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
    <div className="section section--policies">
      <p className="section__header">Policies</p>

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
          />
        </>
      )}
    </div>
  );
};

export default Policies;
