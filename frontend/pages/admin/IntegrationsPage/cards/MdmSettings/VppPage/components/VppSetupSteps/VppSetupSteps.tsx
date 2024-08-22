import CustomLink from "components/CustomLink";
import React from "react";

const baseClass = "vpp-setup-steps";

interface IVppSetupStepsProps {
  /** This prop is used to display additional setup steps content. We have this
   * because some places that use this component show additional content.
   */
  extendendSteps?: boolean;
}

const VppSetupSteps = ({ extendendSteps = false }: IVppSetupStepsProps) => {
  return (
    <ol className={baseClass}>
      <li>
        <span>1.</span>
        <p>
          Sign in to{" "}
          <CustomLink
            newTab
            url="https://business.apple.com"
            text="Apple Business Manager"
          />
          {extendendSteps && (
            <>
              <br />
              If your organization doesn&apos;t have an account, select{" "}
              <b>Sign up now</b>.
            </>
          )}
        </p>
      </li>
      <li>
        <span>2.</span>
        <p>
          Select your <b>account name</b> at the bottom left of the screen, then
          select <b>Preferences</b>.
        </p>
      </li>
      <li>
        <span>3.</span>
        <p>
          Select <b>Payments and Billings</b> in the menu.
        </p>
      </li>
      <li>
        <span>4.</span>
        <p>
          Under the <b>Content Tokens</b>, download the token for the location
          you want to use.
          {extendendSteps && (
            <>
              <br /> Each token is based on a location in Apple Business
              Manager.
            </>
          )}
        </p>
      </li>
      <li>
        <span>5.</span>
        <p>Upload content token (.vpptoken file) below.</p>
      </li>
    </ol>
  );
};

export default VppSetupSteps;
