import React from "react";
import classnames from "classnames";

import TabNav from "components/TabNav";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";
import TabText from "components/TabText";

import CompanyOwnedEndUserAuthPreview from "../../../../../../../../assets/videos/company-owned-end-user-auth-preview.mp4";
import PersonalEndUserAuthPreview from "../../../../../../../../assets/images/personal-end-user-auth-preview.png";

const baseClass = "end-user-experience-preview";

interface IEndUserExperiencePreviewProps {
  className?: string;
}

const EndUserExperiencePreview = ({
  className,
}: IEndUserExperiencePreviewProps) => {
  const classes = classnames(baseClass, className);

  return (
    <div className={classes}>
      <h3>End user experience</h3>
      <TabNav className={`${baseClass}__tab-nav`} secondary>
        <Tabs>
          <TabList>
            <Tab>
              <TabText>Company-owned devices</TabText>
            </Tab>
            <Tab>
              <TabText>Personal devices</TabText>
            </Tab>
          </TabList>
          <TabPanel>
            <p>
              When the end user reaches the <b>Remote Management</b> screen,
              they are first asked to authenticate and agree to the end user
              license agreement (EULA).
            </p>
            {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
            <video
              className={`${baseClass}__preview-video`}
              src={CompanyOwnedEndUserAuthPreview}
              controls
              autoPlay
              loop
              muted
            />
          </TabPanel>
          <TabPanel>
            <p>
              When the end user visits the link from <b>Hosts &gt; Add hosts</b>
              , they are first asked to authenticate.
            </p>
            <img
              className={`${baseClass}__personal-preview-img`}
              src={PersonalEndUserAuthPreview}
              alt="Personal End User Authentication Preview"
            />
          </TabPanel>
        </Tabs>
      </TabNav>
    </div>
  );
};

export default EndUserExperiencePreview;
