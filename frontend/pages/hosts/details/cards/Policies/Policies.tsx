import React from "react";

import { IHostPolicy } from "interfaces/policy";
import InfoBanner from "components/InfoBanner";
import TableContainer from "components/TableContainer";
import CustomLink from "components/CustomLink";
import EmptyTable from "components/EmptyTable";

import {
  generatePolicyTableHeaders,
  generatePolicyDataSet,
} from "./HostPoliciesTable/HostPoliciesTableConfig";
import PolicyFailingCount from "./HostPoliciesTable/PolicyFailingCount";
import { isValidPolicyResponse } from "../../../ManageHostsPage/helpers";

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
              {" "}
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
            <PolicyFailingCount policyList={policies} deviceUser={deviceUser} />
          )}
          {noResponses?.length > 0 && !deviceUser && (
            <InfoBanner>
              <p>
                This host is not updating the response for some policies. Check
                out the Fleet documentation on&nbsp;
                <CustomLink
                  url="https://fleetdm.com/docs/using-fleet/faq#why-is-my-host-not-updating-a-policys-response"
                  text="why the response might not be updating"
                  newTab
                  multiline
                />
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
          />
        </>
      )}
    </div>
  );
};

export default Policies;
