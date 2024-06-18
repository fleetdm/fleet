import React, { useCallback, useContext, useState } from "react";

import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import SectionHeader from "components/SectionHeader";
import TooltipWrapper from "components/TooltipWrapper";
import EditTeamModal from "../EditTeamModal";

const baseClass = "default-team-section";

const DefaultTeamSection = () => {
  const { config } = useContext(AppContext);
  const [showEditTeamModal, setShowEditTeamModal] = useState(false);

  const toggleEditTeamModal = useCallback(() => {
    setShowEditTeamModal((prev) => !prev);
  }, []);

  const defaultTeamName = config?.mdm?.apple_bm_default_team || "No team";

  return (
    <div className={`${baseClass}`}>
      <SectionHeader title="Default team" />
      <p>macOS hosts automatically enroll to this team.</p>
      <h4>
        <TooltipWrapper
          position="top-start"
          tipContent="macOS hosts will be added to this team when they’re first unboxed."
        >
          Team
        </TooltipWrapper>
      </h4>
      <p>
        {config?.mdm?.apple_bm_default_team || "No team"}{" "}
        <Button
          className={`${baseClass}__edit-team-btn`}
          onClick={toggleEditTeamModal}
          variant="text-icon"
        >
          Edit <Icon name="pencil" />
        </Button>
      </p>
      {showEditTeamModal && (
        <EditTeamModal
          defaultTeamName={defaultTeamName}
          onCancel={toggleEditTeamModal}
        />
      )}
    </div>
  );
};

export default DefaultTeamSection;
