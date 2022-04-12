import React from "react";

import { IHostPolicy } from "interfaces/policy";
import InfoBanner from "components/InfoBanner";
import TableContainer from "components/TableContainer";
import {
  generatePolicyTableHeaders,
  generatePolicyDataSet,
} from "./HostPoliciesTable/HostPoliciesTableConfig";
import PolicyFailingCount from "./HostPoliciesTable/PolicyFailingCount";
import { isValidPolicyResponse } from "../../../ManageHostsPage/helpers";

interface IPoliciesProps {
  policies: IHostPolicy[];
  isLoading: boolean;
  togglePolicyDetailsModal: (policy: IHostPolicy) => void;
}

const Policies = ({
  policies,
  isLoading,
  togglePolicyDetailsModal,
}: IPoliciesProps): JSX.Element => {
  if (policies.length === 0) {
    return (
      <div className="section section--policies">
        <p className="section__header">Policies</p>
        <div className="results__data">
          <b>No policies are checked for this host.</b>
          <p>
            Expecting to see policies? Try selecting “Refetch” to ask this host
            to report new vitals.
          </p>
        </div>
      </div>
    );
  }

  const tableHeaders = generatePolicyTableHeaders(togglePolicyDetailsModal);
  const noResponses: IHostPolicy[] =
    policies.filter(
      (policy: IHostPolicy) => !isValidPolicyResponse(policy.response)
    ) || [];
  const failingResponses: IHostPolicy[] =
    policies.filter((policy: IHostPolicy) => policy.response === "fail") || [];

  return (
    <div className="section section--policies">
      <p className="section__header">Policies</p>

      {policies.length > 0 && (
        <>
          {failingResponses?.length > 0 && (
            <PolicyFailingCount policyList={policies} />
          )}
          {noResponses?.length > 0 && (
            <InfoBanner>
              <p>
                This host is not updating the response for some policies. Check
                out the Fleet documentation on&nbsp;
                <a
                  href="https://fleetdm.com/docs/using-fleet/faq#why-my-host-is-not-updating-a-policys-response"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  why the response might not be updating
                </a>
                .
              </p>
            </InfoBanner>
          )}
          <TableContainer
            columns={tableHeaders}
            data={generatePolicyDataSet(policies)}
            isLoading={isLoading}
            defaultSortHeader={"name"}
            defaultSortDirection={"asc"}
            resultsTitle={"policy items"}
            emptyComponent={() => <></>}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            disablePagination
            disableCount
            highlightOnHover
          />
        </>
      )}
    </div>
  );
};

export default Policies;
