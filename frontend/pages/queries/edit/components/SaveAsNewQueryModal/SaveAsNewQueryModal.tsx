import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";

import { getPathWithQueryParams } from "utilities/url";

import { ICreateQueryRequestBody } from "interfaces/schedulable_query";

import queryAPI from "services/entities/queries";
import { NotificationContext } from "context/notification";

import { getErrorReason, IApiError } from "interfaces/errors";
import {
  INVALID_PLATFORMS_FLASH_MESSAGE,
  INVALID_PLATFORMS_REASON,
} from "utilities/constants";
import { create } from "lodash";
import { ITeam, ITeamSummary } from "interfaces/team";
import Modal from "components/Modal";

const baseClass = "save-as-new-query-modal";

interface ISaveAsNewQueryModal {
  router: InjectedRouter;
  initialQueryData: ICreateQueryRequestBody;
  onExit: () => void;
}

interface ISANQFormData {
  name: string;
  team: Partial<ITeamSummary>;
}

const SaveAsNewQueryModal = ({
  router,
  initialQueryData,
  onExit,
}: ISaveAsNewQueryModal) => {
  const { renderFlash } = useContext(NotificationContext);

  const [formData, setFormData] = useState<ISANQFormData>({
    name: `Copy of ${initialQueryData.name}`,
    team: {
      id: initialQueryData.team_id,
      name: undefined,
    },
  });

  const [isSaving, setIsSaving] = useState(false);
  // TODO - error state, 1 field
  // TODO - validation

  // take all existing data for query from parent, allow editing name and team
  const handleSave = () => async (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    setIsSaving(true);
    const {
      name: queryName,
      team: { id: teamId, name: teamName },
    } = formData;
    const createBody = {
      ...initialQueryData,
      name: queryName,
      team_id: teamId,
    };
    try {
      const response = await queryAPI.create(createBody);
      setIsSaving(false);
      renderFlash("success", `Successfully added query ${response.name}.`);
      router.push(
        getPathWithQueryParams(PATHS.QUERY_DETAILS(response.query.id), {
          team_id: response.query.team_id,
        })
      );
    } catch (createError: unknown) {
      // { data: IApiError }
      let errFlash = "Could not create query. Please try again.";
      const reason = getErrorReason(createError);
      if (reason.includes("already exists")) {
        let teamText;
        if (teamId !== 0) {
          teamText = teamName ? `the ${teamName} team` : "this team";
        } else {
          teamText = "all teams";
        }
        errFlash = `A query called "${queryName}" already exists for ${teamText}.`;
      } else if (reason.includes(INVALID_PLATFORMS_REASON)) {
        errFlash = INVALID_PLATFORMS_FLASH_MESSAGE;
      }
      setIsSaving(false);
      renderFlash("error", errFlash);
    }
  };
  return (
    <Modal title="Save as new" onExit={onExit}>
      <></>
    </Modal>
  );
};

export default SaveAsNewQueryModal;
