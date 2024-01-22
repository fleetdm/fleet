import TooltipWrapper from "components/TooltipWrapper";
import React from "react";

import { secondsToHms } from "utilities/helpers";

const baseClass = "agent-options";
interface IAgentOptionsProps {
  osqueryData: { [key: string]: any };
  wrapFleetHelper: (helperFn: (value: any) => string, value: string) => string;
  isChromeOS?: boolean;
}

const CHROMEOS_AGENT_OPTIONS = ["Not supported", "Not supported", "10 secs"];
const CHROMEOS_AGENT_OPTIONS_TOOLTIP_MESSAGE =
  "Chromebooks ignore Fleet’s agent options configuration. The options displayed below are the same for all Chromebooks.";
const AgentOptions = ({
  osqueryData,
  wrapFleetHelper,
  isChromeOS = false,
}: IAgentOptionsProps): JSX.Element => {
  let configTLSRefresh;
  let loggerTLSPeriod;
  let distributedInterval;

  if (isChromeOS) {
    [
      configTLSRefresh,
      loggerTLSPeriod,
      distributedInterval,
    ] = CHROMEOS_AGENT_OPTIONS;
  } else {
    configTLSRefresh = wrapFleetHelper(
      secondsToHms,
      osqueryData.config_tls_refresh
    );
    loggerTLSPeriod = wrapFleetHelper(
      secondsToHms,
      osqueryData.logger_tls_period
    );
    distributedInterval = wrapFleetHelper(
      secondsToHms,
      osqueryData.distributed_interval
    );
  }

  return (
    <div className={`${baseClass} section osquery col-50`}>
      {isChromeOS ? (
        <TooltipWrapper
          tipContent={CHROMEOS_AGENT_OPTIONS_TOOLTIP_MESSAGE}
          className="section__header"
        >
          Agent options
        </TooltipWrapper>
      ) : (
        <p className="section__header">Agent options</p>
      )}
      <div className="info-grid">
        <div className="info-grid__block">
          <span className="info-grid__header">Config TLS refresh</span>
          <span className={`info-grid__data ${isChromeOS ? "grey-text" : ""}`}>
            {configTLSRefresh}
          </span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Logger TLS period</span>
          <span className={`info-grid__data ${isChromeOS ? "grey-text" : ""}`}>
            {loggerTLSPeriod}
          </span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Distributed interval</span>
          <span className="info-grid__data">{distributedInterval}</span>
        </div>
      </div>
    </div>
  );
};

export default AgentOptions;
