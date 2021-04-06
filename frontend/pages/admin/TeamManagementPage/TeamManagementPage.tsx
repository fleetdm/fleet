import React from 'react';

const baseClass = 'team-management';

const TeamManagementPage = () => {
  return (
    <div className={`${baseClass} body-wrap`}>
      <p className={`${baseClass}__page-description`}>
        Create, customize, and remove teams from Fleet.
      </p>
    </div>
  );
};

export default TeamManagementPage;
