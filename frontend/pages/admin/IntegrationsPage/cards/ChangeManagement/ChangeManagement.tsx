import SectionHeader from "components/SectionHeader";
import React from "react";

const baseClass = "change-management";

// interface IChangeManagement {

// }
// {}: IChangeManagement
const ChangeManagement = () => {
  return (
    <>
      {/* <div className={`${baseClass}`}> */}
      <SectionHeader title="Calendars" />
      <p className={`${baseClass}__page-description`}>
        To create calendar events for end users with failing policies,
        you&apos;ll need to configure a dedicated Google Workspace service
        account.
      </p>
      {/* </div> */}
    </>
  );
};

export default ChangeManagement;
