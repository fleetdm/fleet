/**
 * Component when there is no schedule set up in fleet
 */
import React from "react";
import { useDispatch } from "react-redux";
import { push } from "react-router-redux";
import paths from "router/paths";

import Button from "components/buttons/Button";
// @ts-ignore
import scheduleSvg from "../../../../../../assets/images/schedule.svg";

const baseClass = "no-schedule";

interface INoScheduleProps {
  toggleScheduleEditorModal: () => void;
}

const NoSchedule = ({
  toggleScheduleEditorModal,
}: INoScheduleProps): JSX.Element => {
  const dispatch = useDispatch();
  const { MANAGE_PACKS } = paths;

  const handleAdvanced = () => dispatch(push(MANAGE_PACKS));

  console.log("noschedule", typeof toggleScheduleEditorModal);
  console.log("toggleScheduleEditorModal", toggleScheduleEditorModal);

  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__inner`}>
        <img src={scheduleSvg} alt="No Schedule" />
        <div className={`${baseClass}__inner-text`}>
          <h2>You don&apos;t have any queries scheduled.</h2>
          <p>
            Schedule a query, or go to your osquery packs via the
            &lsquo;Advanced&rsquo; button.
          </p>
          <div className={`${baseClass}__no-schedule-cta-buttons`}>
            <Button
              variant="brand"
              className={`${baseClass}__schedule-button`}
              onClick={toggleScheduleEditorModal}
            >
              Schedule a query
            </Button>
            <Button
              variant="inverse"
              onClick={handleAdvanced}
              className={`${baseClass}__advanced-button`}
            >
              Advanced
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default NoSchedule;
