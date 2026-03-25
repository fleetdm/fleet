import React from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import { AxiosError } from "axios";

import { internationalTimeFormat } from "utilities/helpers";
import {
  LEARN_MORE_ABOUT_BASE_LINK,
  MDM_STATUS_TOOLTIP,
} from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";
import paths from "router/paths";
import {
  MdmEnrollmentStatus,
  MDM_ENROLLMENT_STATUS_UI_MAP,
} from "interfaces/mdm";
import hostAPI, { IDepAssignmentHostResponse } from "services/entities/hosts";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import Icon from "components/Icon";
import CustomLink from "components/CustomLink";
import List from "components/List";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import TooltipWrapper from "components/TooltipWrapper";
import { IconNames } from "components/icons";

const baseClass = "mdm-status-modal";

interface IMDMStatusModal {
  hostId: number;
  enrollmentStatus: MdmEnrollmentStatus;
  depProfileError?: boolean;
  router: InjectedRouter;
  isPremiumTier?: boolean;
  isMacOSHost?: boolean;
  onExit: () => void;
}

type ProfileStatusCode = "" | "removed" | "assigned" | "pushed";

const PROFILE_STATUS_UI_MAP: Record<
  ProfileStatusCode,
  { label: string; tooltip: JSX.Element | string }
> = {
  "": {
    label: "Empty",
    tooltip: "No profile assigned to this host.",
  },
  removed: {
    label: "Removed",
    tooltip: "Profile was removed from this host.",
  },
  assigned: {
    label: "Assigned",
    tooltip: (
      <>
        Profile is assigned in ABM, and ABM <br />
        is preparing to push it to the host.
      </>
    ),
  },
  pushed: {
    label: "Pushed",
    tooltip: "Profile has been delivered to the host.",
  },
};

const getProfileStatusUI = (raw?: string | null) => {
  const label = (raw ?? "") as ProfileStatusCode;
  return PROFILE_STATUS_UI_MAP[label] ?? PROFILE_STATUS_UI_MAP[""];
};

const PROFILE_ASSIGNMENT_ERROR_UI_MAP: Record<
  string,
  { label: JSX.Element | string; tooltip: JSX.Element | string }
> = {
  throttled: {
    label: "Throttled",
    tooltip: (
      <>
        Migration or new Mac setup won&apos;t work. Fleet hit Apple&apos;s API
        rate limit when preparing the macOS Setup Assistant for this host. Fleet
        will try again in 10 hours.
      </>
    ),
  },
  failed: {
    label: "Failed",
    tooltip: (
      <>
        Migration or new Mac setup won&apos;t work. Apple&apos;s servers
        rejected the request to assign a profile to a host. Fleet will try again
        every hour.
      </>
    ),
  },
  not_accessible: {
    label: "Not accessible",
    tooltip: (
      <>
        Migration or new Mac setup won&apos;t work. Details are not accessible
        from Apple Business Manager (ABM). Verify the host is assigned to your
        MDM server and Fleet has access permissions.
      </>
    ),
  },
};

const getProfileAssignmentError = (raw?: string) => {
  return raw ? PROFILE_ASSIGNMENT_ERROR_UI_MAP[raw] : undefined;
};

interface IStatusRowItem {
  id: string;
  name: string;
  status: MdmEnrollmentStatus;
}

interface IProfileRowItem {
  id: string;
  name: string;
  nameTooltip?: JSX.Element | string;
  status: string;
  statusIconName?: IconNames;
  statusTooltip?: JSX.Element | string;
}

const MDMStatusModal = ({
  hostId,
  enrollmentStatus,
  depProfileError = false,
  isPremiumTier = false,
  isMacOSHost = false,
  router,
  onExit,
}: IMDMStatusModal) => {
  const {
    data: depAssignmentData,
    isLoading: isLoadingDepAssignment,
    isError: isDepAssignmentError,
  } = useQuery<IDepAssignmentHostResponse, AxiosError>(
    ["dep-assignment", hostId],
    () => hostAPI.getDepAssignment(hostId),
    {
      refetchOnWindowFocus: false,
      refetchOnReconnect: false,
      retry: false,
    }
  );

  // Keeping this as-is per your note.
  const fakeDepAssignmentData: IDepAssignmentHostResponse = {
    id: 32,
    dep_device: {
      asset_tag: "",
      color: "MIDNIGHT",
      description: "IPHONE 13 MIDNIGHT 128GB-USA",
      device_assigned_by: "fleetie@example.com",
      device_assigned_date: "2026-01-29T21:17:25Z",
      device_family: "iPhone",
      os: "iOS",
      profile_status: "assigned",
      profile_assign_time: "2026-01-29T21:17:25Z",
      profile_push_time: "2026-01-03T00:00:00Z",
      profile_uuid: "762C4D36550103CCC53AA212A8D31CDD",
      mdm_migration_deadline: null,
      serial_number: "ABC1FND0ZX",
    },
    host_dep_assignment: {
      assign_profile_response: "SUCCESS",
      profile_uuid: "762C4D36550103CCC53AA212A8D31CDD",
      response_updated_at: "2025-12-04 01:35:27",
      added_at: "2025-12-04 01:35:27",
      deleted_at: null,
      abm_token_id: 1,
      mdm_migration_deadline: "2025-12-05 00:00:00.000000",
      mdm_migration_completed: "2025-12-05 00:00:00.000000",
    },
  };

  const enrollmentFilterValue =
    MDM_ENROLLMENT_STATUS_UI_MAP[enrollmentStatus].filterValue;

  const handleClickStatusRow = (item: IStatusRowItem) => {
    const path = getPathWithQueryParams(paths.MANAGE_HOSTS, {
      mdm_enrollment_status: enrollmentFilterValue,
    });
    router.push(path);
    return item;
  };

  const handleClickProfileRow = (item: IProfileRowItem) => {
    const path = getPathWithQueryParams(paths.MANAGE_HOSTS, {
      dep_profile_error: "true",
    });
    router.push(path);
    return item;
  };

  const renderStatusRow = (item: IStatusRowItem) => {
    const statusTooltip = MDM_STATUS_TOOLTIP[item.status];

    console.log("statusTooltip", statusTooltip);
    return (
      <>
        <div className={`${baseClass}__status`}>
          <div className={`${baseClass}__status-title`}>{item.name}</div>
          <div className={`${baseClass}__status-value`}>
            {statusTooltip ? (
              <TooltipWrapper tipContent={MDM_STATUS_TOOLTIP[item.status]}>
                {MDM_ENROLLMENT_STATUS_UI_MAP[item.status].displayName}
              </TooltipWrapper>
            ) : (
              MDM_ENROLLMENT_STATUS_UI_MAP[item.status].displayName
            )}
          </div>
        </div>
        <ViewAllHostsLink
          queryParams={{ mdm_enrollment_status: enrollmentFilterValue }}
          rowHover
          noLink
        />
      </>
    );
  };

  const renderProfileRow = (item: IProfileRowItem) => (
    <>
      <div className={`${baseClass}__status`}>
        <div className={`${baseClass}__status-title`}>
          {item.nameTooltip ? (
            <TooltipWrapper tipContent={item.nameTooltip}>
              {item.name}
            </TooltipWrapper>
          ) : (
            item.name
          )}
        </div>
        <div className={`${baseClass}__status-value`}>
          {item.statusIconName && <Icon name={item.statusIconName} />}
          {item.statusTooltip ? (
            <TooltipWrapper tipContent={item.statusTooltip}>
              {item.status}
            </TooltipWrapper>
          ) : (
            item.status
          )}
        </div>
      </div>
      <ViewAllHostsLink
        queryParams={{ mdm_enrollment_status: item.status }}
        rowHover
        noLink
      />
    </>
  );

  const renderMDMStatus = () => {
    const data: IStatusRowItem[] = [
      {
        id: "mdm-status",
        name: "MDM status",
        status: enrollmentStatus,
      },
    ];

    return (
      <List<IStatusRowItem>
        data={data}
        renderItemRow={renderStatusRow}
        onClickRow={handleClickStatusRow}
      />
    );
  };

  const renderProfileAssignmentList = () => {
    const data: IProfileRowItem[] = [
      {
        id: "profile-assigned",
        name: "Profile assigned",
        nameTooltip: (
          <>
            The last time Apple reported a profile was assigned
            <br />
            to this host in Apple Business Manager.
          </>
        ),
        status: internationalTimeFormat(
          new Date(fakeDepAssignmentData.dep_device.profile_assign_time)
        ),
      },
      {
        id: "profile-pushed",
        name: "Profile pushed",
        nameTooltip: (
          <>
            The last time Apple reported the host retrieved its <br />
            assigned profile. If a profile wasn&apos;t pushed, the <br />
            host won&apos;t be able to turn on MDM.
          </>
        ),
        status: internationalTimeFormat(
          new Date(fakeDepAssignmentData.dep_device.profile_push_time)
        ),
      },
      {
        id: "profile-status",
        name: "Profile status",
        status: getProfileStatusUI(
          fakeDepAssignmentData.dep_device.profile_status
        ).label,
        statusTooltip: getProfileStatusUI(
          fakeDepAssignmentData.dep_device.profile_status
        ).tooltip,
      },
    ];

    if (depProfileError) {
      const assignmentError = getProfileAssignmentError(
        depAssignmentData?.host_dep_assignment.assign_profile_response
      );

      if (assignmentError) {
        data.push({
          id: "profile-error",
          name: "Profile assignment error",
          status: String(assignmentError.label),
          statusIconName: "error",
          statusTooltip: assignmentError.tooltip,
        });
      }
    }

    if (isLoadingDepAssignment) {
      return <Spinner />;
    }

    if (isDepAssignmentError) {
      return (
        <DataError description="We can't retrieve data from Apple right now. Please try again later." />
      );
    }

    return (
      <List<IProfileRowItem>
        data={data}
        renderItemRow={renderProfileRow}
        onClickRow={handleClickProfileRow}
      />
    );
  };

  const renderProfileAssignment = () => {
    return (
      <div className={`${baseClass}__profile-assignment`}>
        <p>
          <b>Profile assignment</b>
        </p>
        <p>
          Details about automatic enrollment profile from Apple Business
          Manager.{" "}
          <CustomLink
            text="Learn more"
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/abm-issues`}
            newTab
          />
        </p>
        {renderProfileAssignmentList()}
      </div>
    );
  };

  const renderFooter = () => (
    <ModalFooter
      primaryButtons={
        <Button type="button" onClick={onExit}>
          Done
        </Button>
      }
    />
  );

  return (
    <Modal title="MDM status" className={baseClass} onExit={onExit}>
      {renderMDMStatus()}
      {isPremiumTier && isMacOSHost && renderProfileAssignment()}
      {renderFooter()}
    </Modal>
  );
};

export default MDMStatusModal;
