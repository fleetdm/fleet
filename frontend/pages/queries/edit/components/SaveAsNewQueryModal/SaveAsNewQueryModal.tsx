import React, { useCallback, useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { Location } from "history";
import { AppContext } from "context/app";

import PATHS from "router/paths";

import { getPathWithQueryParams } from "utilities/url";

import { ICreateQueryRequestBody } from "interfaces/schedulable_query";

import queryAPI from "services/entities/queries";
import { NotificationContext } from "context/notification";

import { getErrorReason } from "interfaces/errors";
import {
  INVALID_PLATFORMS_FLASH_MESSAGE,
  INVALID_PLATFORMS_REASON,
} from "utilities/constants";
import {
  API_ALL_TEAMS_ID,
  APP_CONTEXT_ALL_TEAMS_ID,
  ITeamSummary,
} from "interfaces/team";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import TeamsDropdown from "components/TeamsDropdown";
import { useTeamIdParam } from "hooks/useTeamIdParam";

const baseClass = "save-as-new-query-modal";

interface ISaveAsNewQueryModal {
  router: InjectedRouter;
  location: Location;
  initialQueryData: ICreateQueryRequestBody;
  onExit: () => void;
}

interface ISANQFormData {
  queryName: string;
  team: Partial<ITeamSummary>;
}

interface ISANQFormErrors {
  queryName?: string;
  team?: string;
}

const validateFormData = (formData: ISANQFormData): ISANQFormErrors => {
  const errors: ISANQFormErrors = {};

  if (!formData.queryName || formData.queryName.trim() === "") {
    errors.queryName = "Name must be present";
  }

  return errors;
};

const SaveAsNewQueryModal = ({
  router,
  location,
  initialQueryData,
  onExit,
}: ISaveAsNewQueryModal) => {
  const { renderFlash } = useContext(NotificationContext);
  const { isPremiumTier } = useContext(AppContext);

  const [formData, setFormData] = useState<ISANQFormData>({
    queryName: `Copy of ${initialQueryData.name}`,
    team: {
      id: initialQueryData.team_id,
      name: undefined,
    },
  });

  const [isSaving, setIsSaving] = useState(false);
  const [formErrors, setFormErrors] = useState<ISANQFormErrors>({});

  const { userTeams } = useTeamIdParam({
    router,
    location,
    includeAllTeams: true,
    includeNoTeam: false,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: true,
      observer: false,
      observer_plus: false,
    },
  });

  const onInputChange = useCallback(
    ({
      name,
      value,
    }: {
      name: string;
      value: string | Partial<ITeamSummary>;
    }) => {
      const newFormData = { ...formData, [name]: value };
      setFormData(newFormData);

      const newErrors = validateFormData(newFormData);
      const errsToSet: ISANQFormErrors = {};
      Object.keys(formErrors).forEach((k) => {
        if (k in newErrors) {
          errsToSet[k as keyof ISANQFormErrors] =
            newErrors[k as keyof ISANQFormErrors];
        }
      });

      setFormErrors(errsToSet);
    },
    [formData, formErrors]
  );

  const onInputBlur = () => {
    setFormErrors(validateFormData(formData));
  };

  const onTeamChange = useCallback(
    (teamId: number) => {
      const selectedTeam = userTeams?.find((team) => team.id === teamId);
      setFormData((prevData) => ({
        ...prevData,
        team: {
          id: teamId,
          name: selectedTeam ? selectedTeam.name : undefined,
        },
      }));
    },
    [userTeams]
  );

  // take all existing data for query from parent, allow editing name and team
  const handleSave = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const errors = validateFormData(formData);
    if (Object.keys(errors).length > 0) {
      setFormErrors(errors);
      return;
    }

    setIsSaving(true);
    const {
      queryName,
      team: { id: teamId, name: teamName },
    } = formData;
    const createBody = {
      ...initialQueryData,
      name: queryName,
      team_id: teamId === APP_CONTEXT_ALL_TEAMS_ID ? API_ALL_TEAMS_ID : teamId,
    };
    try {
      const { query: newQuery } = await queryAPI.create(createBody);
      setIsSaving(false);
      renderFlash("success", `Successfully added query ${newQuery.name}.`);
      router.push(
        getPathWithQueryParams(PATHS.QUERY_DETAILS(newQuery.id), {
          team_id: newQuery.team_id,
        })
      );
    } catch (createError: unknown) {
      let errFlash = "Could not create query. Please try again.";
      const reason = getErrorReason(createError);
      if (reason.includes("already exists")) {
        let teamText;
        if (teamId !== APP_CONTEXT_ALL_TEAMS_ID) {
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
      <form onSubmit={handleSave} className={baseClass}>
        <InputField
          name="queryName"
          onChange={onInputChange}
          onBlur={onInputBlur}
          value={formData.queryName}
          error={formErrors.queryName}
          inputClassName={`${baseClass}__name`}
          label="Name"
          autofocus
          ignore1password
          parseTarget
        />
        {isPremiumTier && (userTeams?.length || 0) > 1 && (
          <div className="form-field">
            <div className="form-field__label">Team</div>
            <TeamsDropdown
              asFormField
              currentUserTeams={userTeams || []}
              selectedTeamId={formData.team.id}
              onChange={onTeamChange}
            />
          </div>
        )}
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            className="save-as-new-query"
            isLoading={isSaving}
            disabled={Object.keys(formErrors).length > 0 || isSaving}
          >
            Save
          </Button>
          <Button onClick={onExit} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default SaveAsNewQueryModal;
